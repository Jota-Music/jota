// Package saved holds external playlists kept in the library, so they show up
// on the home shelf alongside the user's own. Entries can be added, listed and
// removed. The store is source-agnostic: the id prefix routes the lookup, so
// Spotify and YouTube playlists share one list.
package saved

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

var (
	bucket   = kv.UseBucket("saved-playlists")
	indexKey = "index"
)

type Service struct {
	mu sync.Mutex
	// dirty marks changes the user made since the last sync, so the manual sync
	// is offered only when it has something to write.
	dirty bool
}

func New() *Service {
	return &Service{}
}

// Pending reports whether the store changed since the last sync.
func (s *Service) Pending() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.dirty
}

// index reads the saved list. Callers must hold mu.
func (s *Service) index() (Section, error) {
	var part Section
	err := bucket.GetObject(indexKey, &part)
	if errors.Is(err, kv.ErrKeyNotFound) || errors.Is(err, kv.ErrNotStarted) {
		return Section{}, nil
	}
	if err != nil {
		return Section{}, err
	}
	return clean(part), nil
}

// write stores the saved list. Callers must hold mu.
func (s *Service) write(part Section) error {
	return bucket.SetObject(indexKey, part)
}

func (s *Service) List() ([]music.PlaylistSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	part, err := s.index()
	if err != nil {
		return nil, err
	}
	list := make([]music.PlaylistSummary, 0, len(part.Items))
	for _, item := range part.Items {
		list = append(list, music.PlaylistSummary{
			Id:       item.Id,
			Name:     item.Name,
			Cover:    item.Cover,
			Covers:   item.Covers,
			Subtitle: item.Subtitle,
			Owner:    item.Owner,
		})
	}
	return list, nil
}

// Add keeps a playlist in the library, keyed by the id its source prefixed.
// Saving the same playlist twice is a no-op.
func (s *Service) Add(playlist music.PlaylistSummary) error {
	if playlist.Id == "" {
		return errors.New("empty playlist id")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	part, err := s.index()
	if err != nil {
		return err
	}
	for _, item := range part.Items {
		if item.Id == playlist.Id {
			return nil
		}
	}
	delete(part.Removed, playlist.Id)
	part.Items = append(part.Items, Entry{
		Id:       playlist.Id,
		Name:     playlist.Name,
		Cover:    playlist.Cover,
		Covers:   playlist.Covers,
		Subtitle: playlist.Subtitle,
		Owner:    playlist.Owner,
		At:       time.Now().UnixMilli(),
	})
	if err := s.write(part); err != nil {
		return err
	}
	s.dirty = true
	return nil
}

func (s *Service) Remove(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	part, err := s.index()
	if err != nil {
		return err
	}
	kept := make([]Entry, 0, len(part.Items))
	for _, item := range part.Items {
		if item.Id != id {
			kept = append(kept, item)
		}
	}
	if part.Removed == nil {
		part.Removed = make(map[string]int64, 1)
	}
	part.Removed[id] = time.Now().UnixMilli()
	part.Items = kept
	if err := s.write(part); err != nil {
		return err
	}
	s.dirty = true
	return nil
}
