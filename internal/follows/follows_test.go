package follows

import (
	"reflect"
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
)

func setup(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)
	return New()
}

// There is one list per install: no account scopes it.
func TestFollowLifecycle(t *testing.T) {
	s := setup(t)

	if err := s.Follow("spotify:artist:abc"); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := s.Follow("spotify:artist:abc"); err != nil {
		t.Fatalf("duplicate follow: %v", err)
	}
	if err := s.Follow("youtube:UC123"); err != nil {
		t.Fatalf("follow youtube: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if want := (Users{"spotify:artist:abc", "youtube:UC123"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v, want %v", got, want)
	}
	if !s.Pending() {
		t.Fatal("following did not mark the store dirty")
	}

	if err := s.Unfollow("SPOTIFY:ARTIST:ABC"); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	got, err = s.List()
	if err != nil {
		t.Fatalf("list after unfollow: %v", err)
	}
	if want := (Users{"youtube:UC123"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("after unfollow = %v, want %v", got, want)
	}
}

func TestMergeSectionHonoursTombstones(t *testing.T) {
	s := setup(t)
	if err := s.Follow("spotify:artist:abc"); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := s.Unfollow("spotify:artist:abc"); err != nil {
		t.Fatalf("unfollow: %v", err)
	}

	remote := Section{Items: []Entry{{Id: "spotify:artist:abc", At: 1}, {Id: "youtube:UC123", At: 2}}}
	if _, _, err := s.MergeSection(remote, Newer, true); err != nil {
		t.Fatalf("merge: %v", err)
	}
	users, _ := s.List()
	if want := (Users{"youtube:UC123"}); !reflect.DeepEqual(users, want) {
		t.Fatalf("after merge = %v, want %v; an unfollow must not be undone", users, want)
	}

	// An explicit restore undoes the local unfollow.
	if _, _, err := s.MergeSection(remote, Restore, true); err != nil {
		t.Fatalf("restore: %v", err)
	}
	users, _ = s.List()
	if want := (Users{"spotify:artist:abc", "youtube:UC123"}); !reflect.DeepEqual(users, want) {
		t.Fatalf("after restore = %v, want %v", users, want)
	}
}

// Following again after an unfollow is newer than the tombstone, so it wins a
// background sync instead of waiting for an explicit restore.
func TestMergeSectionNewerFollowBeatsTombstone(t *testing.T) {
	s := setup(t)
	if err := s.Follow("spotify:artist:abc"); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := s.Unfollow("spotify:artist:abc"); err != nil {
		t.Fatalf("unfollow: %v", err)
	}

	remote := Section{Items: []Entry{{Id: "spotify:artist:abc", At: time.Now().Add(time.Hour).UnixMilli()}}}
	out, _, err := s.MergeSection(remote, Newer, true)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	users, _ := s.List()
	if want := (Users{"spotify:artist:abc"}); !reflect.DeepEqual(users, want) {
		t.Fatalf("after merge = %v, want %v", users, want)
	}
	if len(out.Removed) != 0 {
		t.Fatalf("tombstone survived a newer follow: %v", out.Removed)
	}
}

func TestMergeSectionWithoutApplyLeavesStore(t *testing.T) {
	s := setup(t)
	if err := s.Follow("spotify:artist:abc"); err != nil {
		t.Fatalf("follow: %v", err)
	}

	remote := Section{Items: []Entry{{Id: "youtube:UC123", At: 2}}}
	out, applied, err := s.MergeSection(remote, Newer, false)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	if applied != 0 {
		t.Fatalf("applied = %d, want 0 without apply", applied)
	}
	if want := (Users{"spotify:artist:abc", "youtube:UC123"}); !reflect.DeepEqual(users(out), want) {
		t.Fatalf("union = %v, want %v", users(out), want)
	}
	got, _ := s.List()
	if want := (Users{"spotify:artist:abc"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("store = %v, want %v untouched", got, want)
	}
}

// users lists the ids in the order the union answers them.
func users(part Section) Users {
	list := make(Users, 0, len(part.Items))
	for _, item := range part.Items {
		list = append(list, item.Id)
	}
	return list
}
