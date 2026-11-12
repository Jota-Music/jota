package playlists

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/state"
)

// ErrNotFound is what a caller gets when a playlist is not in the store.
var ErrNotFound = errors.New("playlist not found")

type Service struct {
	mu            sync.Mutex
	resolve       func(id string) (music.Song, error)
	trackCache    func(id string) (music.Song, bool)
	setTrackCache func(id string, song music.Song)
	// dirty marks what a pass still has to write: a playlist the user changed, or
	// metadata resolved here that the document does not carry yet.
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
	return music.LocalPrefix + "playlist:" + hex.EncodeToString(buf), nil
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

// stored reads everything the device keeps of the local playlists.
func (s *Service) stored() (Stored, error) {
	return state.Read[Stored](state.Playlists)
}

func (s *Service) load(id string) (Playlist, error) {
	part, err := s.stored()
	if err != nil {
		return Playlist{}, err
	}
	pl, ok := part.Tracks[id]
	if !ok {
		return Playlist{}, ErrNotFound
	}
	return pl, nil
}

func (s *Service) save(pl Playlist) error {
	part, err := s.stored()
	if err != nil {
		return err
	}
	if part.Tracks == nil {
		part.Tracks = map[string]Playlist{}
	}
	part.Tracks[pl.Id] = pl
	return state.Write(state.Playlists, part)
}

func (s *Service) index() ([]string, error) {
	part, err := s.stored()
	if err != nil {
		return nil, err
	}
	return part.Index, nil
}

func (s *Service) setIndex(ids []string) error {
	part, err := s.stored()
	if err != nil {
		return err
	}
	part.Index = ids
	return state.Write(state.Playlists, part)
}

// tombstones maps a deleted playlist id to the moment it was deleted, so the
// backup file cannot bring it back.
func (s *Service) tombstones() (map[string]int64, error) {
	part, err := s.stored()
	if err != nil {
		return nil, err
	}
	if part.Deleted == nil {
		return map[string]int64{}, nil
	}
	return part.Deleted, nil
}

func (s *Service) bury(id string, at int64) error {
	part, err := s.stored()
	if err != nil {
		return err
	}
	if part.Deleted == nil {
		part.Deleted = map[string]int64{}
	}
	part.Deleted[id] = at
	return state.Write(state.Playlists, part)
}
