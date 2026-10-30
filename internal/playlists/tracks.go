package playlists

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

// GetFullPlaylist implements music.Source: it serves the metadata stored with
// each reference and resolves only the ones it does not have. A reference that
// no longer resolves is marked Broken instead of failing the whole playlist.
func (s *Service) GetFullPlaylist(id string) (music.Playlist, error) {
	pl, err := s.load(id)
	if err != nil {
		if errors.Is(err, kv.ErrKeyNotFound) {
			return music.Playlist{}, fmt.Errorf("playlist %q not found", id)
		}
		return music.Playlist{}, err
	}

	songs := make([]music.Song, 0, len(pl.Songs))
	known := make(map[string]music.Song, len(pl.Songs))
	for _, ref := range pl.Songs {
		// The metadata was persisted when the reference was added, so a playlist
		// opens from the store instead of resolving every track again: on a
		// restored library that resolve is a network round trip per song.
		if stored, ok := pl.Tracks[ref]; ok {
			songs = append(songs, stored)
			continue
		}

		var song music.Song
		err := errors.New("no resolver")
		if s.resolve != nil {
			song, err = s.resolve(ref)
		}
		if err != nil {
			broken := music.Song{Id: ref, Broken: true}
			if s.trackCache != nil {
				if cached, ok := s.trackCache(ref); ok {
					broken = cached
					broken.Broken = true
				}
			}
			if broken.Id == "" {
				broken.Id = ref
			}
			songs = append(songs, broken)
			continue
		}
		songs = append(songs, song)
		known[ref] = song
	}

	if len(known) > 0 {
		s.rememberTracks(id, known)
	}

	return music.Playlist{Name: pl.Name, Songs: songs}, nil
}

// rememberTracks persists resolved metadata so the playlist stays readable
// offline and its backup remains self-contained. Storing it marks the playlist
// as work to write, or the document would keep exporting bare ids.
func (s *Service) rememberTracks(id string, tracks map[string]music.Song) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fresh, err := s.load(id)
	if err != nil {
		return
	}
	if fresh.Tracks == nil {
		fresh.Tracks = make(map[string]music.Song, len(tracks))
	}
	changed := false
	for ref, song := range tracks {
		if previous, ok := fresh.Tracks[ref]; ok && reflect.DeepEqual(previous, song) {
			continue
		}
		fresh.Tracks[ref] = song
		changed = true
	}
	if changed {
		// The metadata has to travel: a playlist exported as bare ids makes the
		// other device ask the source for every track it draws, and nothing else
		// would ever write it out.
		s.dirty = true
		_ = s.save(fresh)
	}
}

func (s *Service) RevalidateFullPlaylist(id string) error {
	return nil
}

func (s *Service) GetSong(id string) (music.Song, error) {
	return music.Song{}, errors.New("local playlists are not a song source")
}

// hydrateTimeout bounds a whole hydration pass: a stalled source must never
// hold an export hostage.
const hydrateTimeout = 10 * time.Second

// hydrate resolves metadata for stored refs that have none yet, so the next
// export is self-contained. Resolving hits the network, so it runs outside the
// lock and only refs that are actually missing are fetched.
func (s *Service) hydrate() {
	s.mu.Lock()
	ids, err := s.index()
	resolve := s.resolve
	pending := make(map[string][]string)
	if err == nil && resolve != nil {
		for _, id := range ids {
			pl, err := s.load(id)
			if err != nil {
				continue
			}
			for _, ref := range pl.Songs {
				if _, ok := pl.Tracks[strings.TrimSpace(ref)]; ok {
					continue
				}
				pending[id] = append(pending[id], ref)
			}
		}
	}
	s.mu.Unlock()

	deadline := time.Now().Add(hydrateTimeout)
	for id, refs := range pending {
		known := make(map[string]music.Song, len(refs))
		for _, ref := range refs {
			left := time.Until(deadline)
			if left <= 0 {
				break
			}
			if song, ok := resolveWithin(resolve, ref, left); ok {
				known[ref] = song
			}
		}
		if len(known) > 0 {
			s.rememberTracks(id, known)
		}
	}
}

// ponytail: a timed-out resolve leaks its goroutine until the source answers;
// every source already enforces its own HTTP timeout, so this is bounded.
func resolveWithin(resolve func(string) (music.Song, error), ref string, timeout time.Duration) (music.Song, bool) {
	type result struct {
		song music.Song
		err  error
	}
	done := make(chan result, 1)
	go func() {
		song, err := resolve(ref)
		done <- result{song, err}
	}()
	select {
	case res := <-done:
		return res.song, res.err == nil
	case <-time.After(timeout):
		return music.Song{}, false
	}
}
