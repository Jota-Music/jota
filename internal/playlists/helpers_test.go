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
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)
}

func seed(t *testing.T, s *Service, playlists []exportPlaylist) {
	t.Helper()
	ids, err := s.index()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range playlists {
		pl := Playlist{Id: p.Id, Name: p.Name, Songs: p.Songs, Tracks: p.Tracks, ModifiedAt: p.ModifiedAt}
		if err := s.save(pl); err != nil {
			t.Fatal(err)
		}
		if !slices.Contains(ids, p.Id) {
			ids = append(ids, p.Id)
		}
	}
	if err := s.setIndex(ids); err != nil {
		t.Fatal(err)
	}
}

// Every precedence rule of the merge lives in one table: how a foreign id maps
// to a stored record, which copy wins by time, what an untimed copy may do, that
// the winning order survives, and that a second pass changes nothing.
