package saved

import (
	"testing"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

func youtubePlaylist(id string) music.PlaylistSummary {
	return music.PlaylistSummary{Id: music.YouTubePrefix + id, Name: id}
}

func spotifyPlaylist(id string) music.PlaylistSummary {
	return music.PlaylistSummary{Id: id, Name: id}
}

func TestSavedLifecycle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)

	s := New()

	list, err := s.List()
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("empty store = %v", list)
	}

	if err := s.Add(youtubePlaylist("PL1")); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := s.Add(spotifyPlaylist("37i9dQ")); err != nil {
		t.Fatalf("add spotify: %v", err)
	}
	// A repeated save must not duplicate the entry.
	if err := s.Add(youtubePlaylist("PL1")); err != nil {
		t.Fatalf("add duplicate: %v", err)
	}
	if err := s.Add(music.PlaylistSummary{}); err == nil {
		t.Fatal("add without id: want error, got nil")
	}

	list, err = s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %v, want the 2 saved playlists", list)
	}
	if list[0].Id != "youtube:PL1" || list[1].Id != "37i9dQ" {
		t.Fatalf("list = %v, want youtube then spotify", list)
	}

	if err := s.Remove("37i9dQ"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	list, err = s.List()
	if err != nil {
		t.Fatalf("list after remove: %v", err)
	}
	if len(list) != 1 || list[0].Id != "youtube:PL1" {
		t.Fatalf("after remove = %v, want only youtube:PL1", list)
	}
}
