package playlists

import (
	"errors"
	"testing"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

type fakeResolver struct {
	fail map[string]bool
}

func (f *fakeResolver) GetSong(id string) (music.Song, error) {
	if f.fail[id] {
		return music.Song{}, errors.New("unavailable")
	}
	return music.Song{Id: id, Name: "Song " + id}, nil
}

func setup(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)
}

func TestPlaylistLifecycle(t *testing.T) {
	setup(t)

	resolver := &fakeResolver{fail: map[string]bool{}}
	s := New(resolver)

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
	resolver.fail["bad"] = true
	if err := s.AddSongs(summary.Id, []string{"ok", "bad"}); err == nil {
		t.Fatal("add with failing ref = nil, want error")
	}
	pl, _ = s.GetFullPlaylist(summary.Id)
	if len(pl.Songs) != 2 {
		t.Fatalf("after failed add = %d songs, want 2", len(pl.Songs))
	}

	// A reference that stops resolving later is marked broken, not dropped.
	resolver.fail["spotifyId"] = true
	pl, _ = s.GetFullPlaylist(summary.Id)
	if !pl.Songs[0].Broken || pl.Songs[0].Id != "spotifyId" {
		t.Fatalf("broken = %+v", pl.Songs[0])
	}
	resolver.fail["spotifyId"] = false

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
