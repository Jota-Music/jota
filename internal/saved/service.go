// Package saved holds external playlists kept in the library, so they show up
// on the home shelf alongside the user's own. Entries can be listed and removed.
package saved

import (
	"errors"
	"sync"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

var (
	bucket   = kv.UseBucket("saved-playlists")
	indexKey = "index"
)

type Service struct {
	mu sync.Mutex
}

func New() *Service {
	return &Service{}
}

// index reads the saved list. Callers must hold mu.
func (s *Service) index() ([]music.PlaylistSummary, error) {
	var list []music.PlaylistSummary
	err := bucket.GetObject(indexKey, &list)
	if errors.Is(err, kv.ErrKeyNotFound) || errors.Is(err, kv.ErrNotStarted) {
		return nil, nil
	}
	return list, err
}

func (s *Service) List() ([]music.PlaylistSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.index()
}

func (s *Service) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.index()
	if err != nil {
		return err
	}

	out := make([]music.PlaylistSummary, 0, len(list))
	for _, p := range list {
		if p.Id != id {
			out = append(out, p)
		}
	}
	return bucket.SetObject(indexKey, out)
}
