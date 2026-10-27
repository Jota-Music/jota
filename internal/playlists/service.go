package playlists

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

var (
	bucket    = kv.UseBucket("playlists")
	indexKey  = "index"
	recordKey = "p:"
	graveKey  = "graves"
)

type Service struct {
	mu            sync.Mutex
	resolve       func(id string) (music.Song, error)
	trackCache    func(id string) (music.Song, bool)
	setTrackCache func(id string, song music.Song)
	// dirty marks playlists the user changed since the last sync, so the manual
	// sync is offered only when it has something to write.
	dirty bool
}

func New(resolve func(id string) (music.Song, error)) *Service {
	return &Service{resolve: resolve}
}

// SetResolver wires the catalog after construction, breaking the cycle between
// the catalog (which routes local playlists) and this service (which resolves
// tracks through the catalog).
func (s *Service) SetResolver(resolve func(id string) (music.Song, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resolve = resolve
}

// SetTrackCache wires best-effort metadata access for Spotify tracks, used to
// keep backups self-contained and to show titles without a live session.
func (s *Service) SetTrackCache(get func(id string) (music.Song, bool), set func(id string, song music.Song)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trackCache = get
	s.setTrackCache = set
}

// Pending reports whether the user changed a playlist since the last sync.
func (s *Service) Pending() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirty
}

func newId() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return music.LocalPrefix + hex.EncodeToString(buf), nil
}

// portable converts a stored playlist into its export shape. Metadata comes
// from the playlist itself, then from the Spotify cache, so a backup stays
// self-contained without a network round trip.
func (s *Service) portable(pl Playlist) exportPlaylist {
	songs := normalizeRefs(pl.Songs)
	tracks := normalizeTracks(pl.Tracks)
	if s.trackCache != nil {
		for _, ref := range songs {
			if _, ok := tracks[ref]; ok {
				continue
			}
			if cached, ok := s.trackCache(ref); ok {
				if tracks == nil {
					tracks = make(map[string]music.Song, len(songs))
				}
				tracks[ref] = cached
			}
		}
	}
	return exportPlaylist{
		Id:         pl.Id,
		Source:     "local",
		Name:       pl.Name,
		Songs:      songs,
		Tracks:     tracks,
		ModifiedAt: pl.ModifiedAt,
	}
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

// tombstones maps a deleted playlist id to the moment it was deleted, so the
// backup file cannot bring it back.
func (s *Service) tombstones() (map[string]int64, error) {
	graves := map[string]int64{}
	if err := bucket.GetObject(graveKey, &graves); err != nil {
		if errors.Is(err, kv.ErrKeyNotFound) || errors.Is(err, kv.ErrNotStarted) {
			return map[string]int64{}, nil
		}
		return nil, err
	}
	return graves, nil
}

func (s *Service) bury(id string, at int64) error {
	graves, err := s.tombstones()
	if err != nil {
		return err
	}
	graves[id] = at
	return bucket.SetObject(graveKey, graves)
}
