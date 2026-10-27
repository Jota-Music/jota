package playlists

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

func (s *Service) List() ([]music.PlaylistSummary, error) {
	ids, err := s.index()
	if err != nil {
		return nil, err
	}

	out := make([]music.PlaylistSummary, 0, len(ids))
	for _, id := range ids {
		pl, err := s.load(id)
		if err != nil {
			if errors.Is(err, kv.ErrKeyNotFound) {
				continue
			}
			return nil, err
		}
		out = append(out, s.summary(pl))
	}
	return out, nil
}

// summary is the listing shape of a stored playlist: its name and the mosaic of
// the covers of its first tracks.
func (s *Service) summary(pl Playlist) music.PlaylistSummary {
	return music.PlaylistSummary{
		Id:     pl.Id,
		Name:   pl.Name,
		Covers: s.covers(pl.Songs),
	}
}

// covers collects the album covers of the first tracks, skipping unresolved
// tracks and duplicate covers so the mosaic never repeats the same art.
func (s *Service) covers(refs []string) []string {
	if s.resolve == nil {
		return nil
	}

	covers := make([]string, 0, coverCount)
	seen := make(map[string]struct{}, coverCount)
	for _, ref := range refs {
		if len(covers) == coverCount {
			break
		}
		song, err := s.resolve(ref)
		if err != nil || len(song.Album.Covers) == 0 {
			continue
		}
		cover := song.Album.Covers[0]
		if _, dup := seen[cover]; dup {
			continue
		}
		seen[cover] = struct{}{}
		covers = append(covers, cover)
	}
	return covers
}

func (s *Service) Create(name string) (music.PlaylistSummary, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultName
	}

	id, err := newId()
	if err != nil {
		return music.PlaylistSummary{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true

	pl := Playlist{Id: id, Name: name, Songs: []string{}, ModifiedAt: time.Now().UnixMilli()}
	if err := s.save(pl); err != nil {
		return music.PlaylistSummary{}, err
	}

	ids, err := s.index()
	if err != nil {
		return music.PlaylistSummary{}, err
	}
	if err := s.setIndex(append(ids, id)); err != nil {
		return music.PlaylistSummary{}, err
	}

	return music.PlaylistSummary{Id: id, Name: name}, nil
}

func (s *Service) Rename(id, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultName
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true

	pl, err := s.load(id)
	if err != nil {
		return err
	}
	pl.Name = name
	pl.ModifiedAt = time.Now().UnixMilli()
	return s.save(pl)
}

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true

	ids, err := s.index()
	if err != nil {
		return err
	}

	out := ids[:0]
	for _, existing := range ids {
		if existing != id {
			out = append(out, existing)
		}
	}
	if err := s.setIndex(out); err != nil {
		return err
	}
	if err := s.bury(id, time.Now().UnixMilli()); err != nil {
		return err
	}
	return bucket.Delete(recordKey + id)
}

// AddSongs resolves every reference before persisting anything, so a playlist
// never stores a reference that cannot be resolved at add time.
func (s *Service) AddSongs(id string, refs []string) error {
	refs = normalizeRefs(refs)
	if len(refs) == 0 {
		return nil
	}

	resolved := make(map[string]music.Song, len(refs))
	for _, ref := range refs {
		song, err := s.resolve(ref)
		if err != nil {
			return fmt.Errorf("cannot add %q: %w", ref, err)
		}
		resolved[ref] = song
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true

	pl, err := s.load(id)
	if err != nil {
		return err
	}

	if pl.Tracks == nil {
		pl.Tracks = make(map[string]music.Song, len(resolved))
	}
	for ref, song := range resolved {
		pl.Tracks[ref] = song
	}
	for _, ref := range refs {
		if !slices.Contains(pl.Songs, ref) {
			pl.Songs = append(pl.Songs, ref)
		}
	}
	pl.ModifiedAt = time.Now().UnixMilli()
	return s.save(pl)
}

func (s *Service) RemoveSong(id string, ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true

	pl, err := s.load(id)
	if err != nil {
		return err
	}

	out := pl.Songs[:0]
	for _, existing := range pl.Songs {
		if existing != ref {
			out = append(out, existing)
		}
	}
	delete(pl.Tracks, ref)
	pl.Songs = out
	pl.ModifiedAt = time.Now().UnixMilli()
	return s.save(pl)
}

// Reorder replaces the playlist order with refs. Unknown references are dropped,
// so a stale frontend cannot corrupt the stored list.
func (s *Service) Reorder(id string, refs []string) error {
	refs = normalizeRefs(refs)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true

	pl, err := s.load(id)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(pl.Songs))
	for _, ref := range pl.Songs {
		seen[ref] = struct{}{}
	}

	out := make([]string, 0, len(refs))
	added := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if _, ok := seen[ref]; !ok {
			continue
		}
		if _, dup := added[ref]; dup {
			continue
		}
		added[ref] = struct{}{}
		out = append(out, ref)
	}
	pl.Songs = out
	pl.ModifiedAt = time.Now().UnixMilli()
	return s.save(pl)
}

// GetPlaylistSummary implements music.Source.
func (s *Service) GetPlaylistSummary(id string) (music.PlaylistSummary, error) {
	pl, err := s.load(id)
	if err != nil {
		return music.PlaylistSummary{}, err
	}
	return s.summary(pl), nil
}
