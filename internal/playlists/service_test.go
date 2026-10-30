package playlists

import (
	"slices"
	"testing"

	"github.com/Jota-Music/jota/internal/music"
)

func TestPlaylistLifecycle(t *testing.T) {
	setup(t)

	fail := map[string]bool{}
	s := New(fakeResolver(nil, fail))

	summary, err := s.Create("Chill")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if summary.Id == "" || summary.Name != "Chill" {
		t.Fatalf("create = %+v", summary)
	}

	list, err := s.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %v, %v", list, err)
	}

	if err := s.AddSongs(summary.Id, []string{"spotifyId", "youtube:vid", "spotifyId"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	list, _ = s.List()
	if want := []string{"cover-spotifyId", "cover-youtube:vid"}; !slices.Equal(list[0].Covers, want) {
		t.Fatalf("covers = %v, want %v", list[0].Covers, want)
	}

	pl, err := s.GetFullPlaylist(summary.Id)
	if err != nil {
		t.Fatalf("full: %v", err)
	}
	if len(pl.Songs) != 2 {
		t.Fatalf("songs = %d, want 2 (dedup)", len(pl.Songs))
	}
	if pl.Songs[0].Name != "Song spotifyId" {
		t.Fatalf("song = %+v", pl.Songs[0])
	}

	// A failing reference rejects the whole add, storing nothing.
	fail["bad"] = true
	if err := s.AddSongs(summary.Id, []string{"ok", "bad"}); err == nil {
		t.Fatal("add with failing ref = nil, want error")
	}
	pl, _ = s.GetFullPlaylist(summary.Id)
	if len(pl.Songs) != 2 {
		t.Fatalf("after failed add = %d songs, want 2", len(pl.Songs))
	}

	// A stored track keeps the metadata it was added with even when the source
	// stops answering: the playlist opens from the store, which is what keeps it
	// readable offline.
	fail["spotifyId"] = true
	pl, _ = s.GetFullPlaylist(summary.Id)
	if pl.Songs[0].Broken || pl.Songs[0].Name != "Song spotifyId" {
		t.Fatalf("stored song = %+v", pl.Songs[0])
	}
	fail["spotifyId"] = false

	// Reorder keeps only known references and preserves the given order.
	if err := s.Reorder(summary.Id, []string{"youtube:vid", "spotifyId", "ghost"}); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	pl, _ = s.GetFullPlaylist(summary.Id)
	if len(pl.Songs) != 2 || pl.Songs[0].Id != "youtube:vid" || pl.Songs[1].Id != "spotifyId" {
		t.Fatalf("after reorder = %+v", pl.Songs)
	}

	if err := s.RemoveSong(summary.Id, "youtube:vid"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	pl, _ = s.GetFullPlaylist(summary.Id)
	if len(pl.Songs) != 1 || pl.Songs[0].Id != "spotifyId" {
		t.Fatalf("after remove = %+v", pl.Songs)
	}

	if err := s.Delete(summary.Id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, _ = s.List()
	if len(list) != 0 {
		t.Fatalf("after delete = %v", list)
	}
}

// A playlist restored from the sync document carries the metadata of every
// reference, so opening it and drawing its mosaic must never ask the source: on
// a device that restored the file the source has no cached entry for these.
func TestStoredTracksSkipTheSource(t *testing.T) {
	setup(t)

	calls := 0
	s := New(func(id string) (music.Song, error) {
		calls++
		return music.Song{Id: id, Name: "live " + id}, nil
	})
	seed(t, s, []exportPlaylist{{
		Id:    "local:restored",
		Name:  "Restored",
		Songs: []string{"a", "b"},
		Tracks: map[string]music.Song{
			"a": {Id: "a", Name: "stored a", Album: music.Album{Covers: []string{"cover-a"}}},
			"b": {Id: "b", Name: "stored b", Album: music.Album{Covers: []string{"cover-b"}}},
		},
	}})

	pl, err := s.GetFullPlaylist("local:restored")
	if err != nil {
		t.Fatalf("full: %v", err)
	}
	if len(pl.Songs) != 2 || pl.Songs[0].Name != "stored a" || pl.Songs[1].Name != "stored b" {
		t.Fatalf("songs = %+v", pl.Songs)
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if want := []string{"cover-a", "cover-b"}; !slices.Equal(list[0].Covers, want) {
		t.Fatalf("covers = %v, want %v", list[0].Covers, want)
	}
	if calls != 0 {
		t.Fatalf("resolver calls = %d, want 0", calls)
	}
}

// Metadata learned here has to reach the document: a playlist whose references
// were never resolved exports bare ids, and the other device then has to ask the
// source for every one of them.
func TestResolvedTracksTravelToTheDocument(t *testing.T) {
	setup(t)

	s := New(fakeResolver(nil, nil))
	seed(t, s, []exportPlaylist{{
		Id:    "local:bare",
		Name:  "Bare",
		Songs: []string{"a"},
	}})

	if _, err := s.GetFullPlaylist("local:bare"); err != nil {
		t.Fatalf("full: %v", err)
	}
	if !s.Pending() {
		t.Fatal("resolving a reference left the playlist clean, so its metadata cannot travel")
	}

	// A pass writes it and clears the mark; opening it again learns nothing new.
	s.dirty = false
	if _, err := s.GetFullPlaylist("local:bare"); err != nil {
		t.Fatalf("full: %v", err)
	}
	if s.Pending() {
		t.Fatal("a playlist whose metadata was already stored marked itself dirty again")
	}
}

func TestListCoversDedupAndCap(t *testing.T) {
	setup(t)

	s := New(fakeResolver(map[string]string{
		"a": "same",
		"b": "same",
		"c": "c",
		"d": "d",
		"e": "e",
		"f": "f",
	}, nil))

	summary, err := s.Create("Mix")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.AddSongs(summary.Id, []string{"a", "b", "c", "d", "e", "f"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// a and b share a cover, so the first four unique covers come from a, c, d, e.
	if want := []string{"same", "c", "d", "e"}; !slices.Equal(list[0].Covers, want) {
		t.Fatalf("covers = %v, want %v", list[0].Covers, want)
	}
}
