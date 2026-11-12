package follows

import (
	"strings"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/state"
)

// Service stores the users, Spotify artists and YouTube channels followed from
// the app. There is one list per install.
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

// load reads the stored list. Callers must hold mu.
func (s *Service) load() (Section, error) {
	list, err := state.Read[Section](state.Follows)
	if err != nil {
		return Section{}, err
	}
	return clean(list), nil
}

func (s *Service) List() (Users, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.load()
	if err != nil {
		return nil, err
	}
	users := make(Users, 0, len(list.Items))
	for _, item := range list.Items {
		users = append(users, item.Id)
	}
	return users, nil
}

func (s *Service) Follow(user string) error {
	user = strings.TrimSpace(user)
	if user == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.load()
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if strings.EqualFold(item.Id, user) {
			return nil
		}
	}
	delete(list.Removed, user)
	list.Items = append(list.Items, Entry{Id: user, At: time.Now().UnixMilli()})
	if err := state.Write(state.Follows, list); err != nil {
		return err
	}
	s.dirty = true
	return nil
}

func (s *Service) Unfollow(user string) error {
	user = strings.TrimSpace(user)
	if user == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	list, err := s.load()
	if err != nil {
		return err
	}
	removed := user
	kept := make([]Entry, 0, len(list.Items))
	for _, item := range list.Items {
		if strings.EqualFold(item.Id, user) {
			removed = item.Id
			continue
		}
		kept = append(kept, item)
	}
	if list.Removed == nil {
		list.Removed = make(map[string]int64, 1)
	}
	list.Removed[removed] = time.Now().UnixMilli()
	list.Items = kept
	if err := state.Write(state.Follows, list); err != nil {
		return err
	}
	s.dirty = true
	return nil
}
