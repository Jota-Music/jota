package playlists

import (
	"errors"
	"slices"
	"testing"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

func fakeResolver(covers map[string]string, fail map[string]bool) func(string) (music.Song, error) {
	return func(id string) (music.Song, error) {
		if fail[id] {
			return music.Song{}, errors.New("unavailable")
		}
		cover := covers[id]
		if cover == "" {
			cover = "cover-" + id
		}
		return music.Song{
			Id:    id,
			Name:  "Song " + id,
			Album: music.Album{Covers: []string{cover}},
		}, nil
	}
}

func setup(t *testing.T) {
	t.Helper()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)
}

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

	// A reference that stops resolving later is marked broken, not dropped.
	fail["spotifyId"] = true
	pl, _ = s.GetFullPlaylist(summary.Id)
	if !pl.Songs[0].Broken || pl.Songs[0].Id != "spotifyId" {
		t.Fatalf("broken = %+v", pl.Songs[0])
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
