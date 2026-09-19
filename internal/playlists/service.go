package playlists

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

var (
	bucket    = kv.UseBucket("playlists")
	indexKey  = "index"
	recordKey = "p:"
)

type Service struct {
	mu       sync.Mutex
	resolver Resolver
}

func New(resolver Resolver) *Service {
	return &Service{resolver: resolver}
}

// SetResolver wires the catalog after construction, breaking the cycle between
// the catalog (which routes local playlists) and this service (which resolves
// tracks through the catalog).
func (s *Service) SetResolver(resolver Resolver) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resolver = resolver
}

func newId() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return music.LocalPrefix + hex.EncodeToString(buf), nil
}

func normalizeRefs(refs []string) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref != "" {
			out = append(out, ref)
		}
	}
	return out
}

func (s *Service) load(id string) (Playlist, error) {
	var pl Playlist
	if err := bucket.GetObject(recordKey+id, &pl); err != nil {
		return Playlist{}, err
	}
	return pl, nil
}

func (s *Service) save(pl Playlist) error {
	return bucket.SetObject(recordKey+pl.Id, pl)
}

func (s *Service) index() ([]string, error) {
	var ids []string
	if err := bucket.GetObject(indexKey, &ids); err != nil {
		if errors.Is(err, kv.ErrKeyNotFound) || errors.Is(err, kv.ErrNotStarted) {
			return nil, nil
		}
		return nil, err
	}
	return ids, nil
}

func (s *Service) setIndex(ids []string) error {
	return bucket.SetObject(indexKey, ids)
}

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
		out = append(out, music.PlaylistSummary{
			Id:     pl.Id,
			Name:   pl.Name,
			Covers: s.covers(pl.Songs),
		})
	}
	return out, nil
}

// covers collects the album covers of the first tracks, skipping unresolved
// tracks and duplicate covers so the mosaic never repeats the same art.
func (s *Service) covers(refs []string) []string {
	if s.resolver == nil {
		return nil
	}

	covers := make([]string, 0, coverCount)
	seen := make(map[string]struct{}, coverCount)
	for _, ref := range refs {
		if len(covers) == coverCount {
			break
		}
		song, err := s.resolver.GetSong(ref)
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

	pl := Playlist{Id: id, Name: name, Songs: []string{}}
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

func (s *Service) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	return bucket.Delete(recordKey + id)
}

// AddSongs resolves every reference before persisting anything, so a playlist
// never stores a reference that cannot be resolved at add time.
func (s *Service) AddSongs(id string, refs []string) error {
	refs = normalizeRefs(refs)
	if len(refs) == 0 {
		return nil
	}

	for _, ref := range refs {
		if _, err := s.resolver.GetSong(ref); err != nil {
			return fmt.Errorf("cannot add %q: %w", ref, err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pl, err := s.load(id)
	if err != nil {
		return err
	}

	for _, ref := range refs {
		if !contains(pl.Songs, ref) {
			pl.Songs = append(pl.Songs, ref)
		}
	}
	return s.save(pl)
}

func (s *Service) RemoveSong(id string, ref string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	pl.Songs = out
	return s.save(pl)
}

// Reorder replaces the playlist order with refs. Unknown references are dropped,
// so a stale frontend cannot corrupt the stored list.
func (s *Service) Reorder(id string, refs []string) error {
	refs = normalizeRefs(refs)

	s.mu.Lock()
	defer s.mu.Unlock()

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
	return s.save(pl)
}

// GetFullPlaylist implements music.Source: it resolves each stored reference to
// its metadata. A reference that no longer resolves is marked Broken instead of
// failing the whole playlist.
func (s *Service) GetFullPlaylist(id string) (music.Playlist, error) {
	pl, err := s.load(id)
	if err != nil {
		if errors.Is(err, kv.ErrKeyNotFound) {
			return music.Playlist{}, fmt.Errorf("playlist %q not found", id)
		}
		return music.Playlist{}, err
	}

	songs := make([]music.Song, 0, len(pl.Songs))
	for _, ref := range pl.Songs {
		song, err := s.resolver.GetSong(ref)
		if err != nil {
			songs = append(songs, music.Song{Id: ref, Broken: true})
			continue
		}
		songs = append(songs, song)
	}

	return music.Playlist{Name: pl.Name, Songs: songs}, nil
}

func (s *Service) RevalidateFullPlaylist(id string) error {
	return nil
}

func (s *Service) GetSong(id string) (music.Song, error) {
	return music.Song{}, errors.New("local playlists are not a song source")
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
