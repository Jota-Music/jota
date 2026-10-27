package playlists

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Jota-Music/jota/internal/kv"
)

// Every precedence rule of the merge lives in one table: how a foreign id maps
// to a stored record, which copy wins by time, what an untimed copy may do, that
// the winning order survives, and that a second pass changes nothing.
func TestMerge(t *testing.T) {
	cases := []struct {
		name    string
		local   []exportPlaylist
		remote  Section
		want    []exportPlaylist
		records []string
		gone    []string
		applied int
		twice   bool
	}{
		{
			name: "derives one stable id for a foreign playlist, once",
			remote: Section{Items: []exportPlaylist{
				{Id: "phone-id", Name: "Phone", Songs: []string{"youtube:b"}, ModifiedAt: 10},
			}},
			want:    []exportPlaylist{{Source: "local", Id: "phone-id", Name: "Phone", Songs: []string{"youtube:b"}, ModifiedAt: 10}},
			records: []string{foreignId("phone-id")},
			applied: 1,
			twice:   true,
		},
		{
			name: "a tombstone beats an older remote copy",
			remote: Section{
				Items:   []exportPlaylist{{Id: "local:gone", Name: "Gone", Songs: []string{"youtube:c"}, ModifiedAt: 10}},
				Deleted: map[string]int64{"local:gone": 30},
			},
			gone:    []string{"local:gone"},
			applied: 0,
		},
		{
			name:  "an untimed copy keeps the local songs",
			local: []exportPlaylist{{Id: "local:aaaa", Name: "Local", Songs: []string{"youtube:x"}, ModifiedAt: 0}},
			remote: Section{Items: []exportPlaylist{
				{Id: "local:aaaa", Name: "Remote", Songs: []string{"youtube:y"}, ModifiedAt: 0},
			}},
			want:    []exportPlaylist{{Source: "local", Id: "local:aaaa", Name: "Local", Songs: []string{"youtube:x"}, ModifiedAt: 0}},
			applied: 0,
		},
		{
			name:  "an untimed remote copy fills an empty local one",
			local: []exportPlaylist{{Id: "local:bbbb", Name: "Empty", ModifiedAt: 0}},
			remote: Section{Items: []exportPlaylist{
				{Id: "local:bbbb", Name: "Empty", Songs: []string{"youtube:a"}, ModifiedAt: 0},
			}},
			want:    []exportPlaylist{{Source: "local", Id: "local:bbbb", Name: "Empty", Songs: []string{"youtube:a"}, ModifiedAt: 0}},
			applied: 1,
			twice:   true,
		},
		{
			name:  "the newer copy wins with its own order",
			local: []exportPlaylist{{Id: "local:cccc", Name: "Mix", Songs: []string{"youtube:a", "youtube:b"}, ModifiedAt: 10}},
			remote: Section{Items: []exportPlaylist{
				{Id: "local:cccc", Name: "Mix", Songs: []string{"youtube:c", "youtube:a", "youtube:b"}, ModifiedAt: 20},
			}},
			want:    []exportPlaylist{{Source: "local", Id: "local:cccc", Name: "Mix", Songs: []string{"youtube:c", "youtube:a", "youtube:b"}, ModifiedAt: 20}},
			applied: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setup(t)
			s := New(nil)
			seed(t, s, c.local)

			got, applied, err := s.merge(c.remote, mergeNewer, true)
			if err != nil {
				t.Fatalf("merge: %v", err)
			}
			if applied != c.applied {
				t.Fatalf("applied = %d, want %d", applied, c.applied)
			}

			want := slices.Clone(c.want)
			slices.SortFunc(want, func(a, b exportPlaylist) int {
				return strings.Compare(a.Id, b.Id)
			})
			if len(got.Items) != len(want) {
				t.Fatalf("playlists = %+v, want %+v", got.Items, want)
			}
			if len(want) > 0 && !reflect.DeepEqual(got.Items, want) {
				t.Fatalf("playlists = %+v, want %+v", got.Items, want)
			}

			for _, id := range c.records {
				if _, err := s.load(id); err != nil {
					t.Fatalf("%s missing from the store: %v", id, err)
				}
			}

			for _, id := range c.gone {
				if _, err := s.load(id); !errors.Is(err, kv.ErrKeyNotFound) {
					t.Fatalf("%s survived a tombstone: %v", id, err)
				}
			}

			if c.twice {
				again, applied, err := s.merge(c.remote, mergeNewer, true)
				if err != nil {
					t.Fatalf("second merge: %v", err)
				}
				if applied != 0 {
					t.Fatalf("second merge applied %d, want 0", applied)
				}
				if !reflect.DeepEqual(again, got) {
					t.Fatalf("second merge changed the document: %+v -> %+v", got, again)
				}
			}
		})
	}
}

// A pass that does not import still answers the union, so the caller can write
// it, but the store keeps its own playlists and its own tombstones.
func TestMergeWithoutApplyLeavesStore(t *testing.T) {
	setup(t)
	s := New(nil)
	own, err := s.Create("Own")
	if err != nil {
		t.Fatal(err)
	}

	remote := Section{
		Items: []exportPlaylist{
			{Id: "phone-id", Name: "Phone", Songs: []string{"youtube:b"}, ModifiedAt: 10},
			{Id: "phone-gone", Name: "Gone", ModifiedAt: 10},
		},
		Deleted: map[string]int64{"phone-gone": 30},
	}
	out, applied, err := s.merge(remote, mergeNewer, false)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if applied != 0 {
		t.Fatalf("applied = %d, want 0 without apply", applied)
	}
	if len(out.Items) != 2 || out.Items[0].Name != "Own" || out.Items[1].Name != "Phone" {
		t.Fatalf("union = %+v, want the local playlist and the one the tombstone lets through", out.Items)
	}
	if _, ok := out.Deleted[storedId("phone-gone")]; !ok {
		t.Fatalf("union = %v, want the tombstone kept", out.Deleted)
	}
	if _, err := s.load(own.Id); err != nil {
		t.Fatalf("store lost its playlist: %v", err)
	}
	if _, err := s.load(foreignId("phone-id")); !errors.Is(err, kv.ErrKeyNotFound) {
		t.Fatalf("store gained the remote playlist: %v", err)
	}
	if _, err := s.load(foreignId("phone-gone")); !errors.Is(err, kv.ErrKeyNotFound) {
		t.Fatalf("store gained the deleted remote playlist: %v", err)
	}
}

// The Android document path and the desktop folder path must end at the same
// document, or a phone and a desktop would keep rewriting each other's backup.
