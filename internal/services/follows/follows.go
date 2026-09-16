package follows

import (
	"errors"
	"strings"
	"sync"

	"github.com/Jota-Music/jota/internal/kv"
)

var bucket = kv.UseBucket("follows")

// Service stores the users followed from the app. Follows are local to Jota
// and scoped by the owning account, so different accounts keep separate lists.
type Service struct {
	mu sync.Mutex
}

func New() *Service {
	return &Service{}
}

func normalizeAccount(account string) string {
	return strings.ToLower(strings.TrimSpace(account))
}

func (s *Service) List(account string) ([]string, error) {
	account = normalizeAccount(account)
	if account == "" {
		return nil, nil
	}

	var users []string
	if err := bucket.GetObject(account, &users); err != nil {
		if errors.Is(err, kv.ErrKeyNotFound) || errors.Is(err, kv.ErrNotStarted) {
			return nil, nil
		}
		return nil, err
	}
	return users, nil
}

func (s *Service) Follow(account string, user string) error {
	account = normalizeAccount(account)
	user = strings.TrimSpace(user)
	if account == "" || user == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	users, err := s.List(account)
	if err != nil {
		return err
	}
	for _, u := range users {
		if strings.EqualFold(u, user) {
			return nil
		}
	}
	return bucket.SetObject(account, append(users, user))
}

func (s *Service) Unfollow(account string, user string) error {
	account = normalizeAccount(account)
	user = strings.TrimSpace(user)
	if account == "" || user == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	users, err := s.List(account)
	if err != nil {
		return err
	}
	out := users[:0]
	for _, u := range users {
		if !strings.EqualFold(u, user) {
			out = append(out, u)
		}
	}
	return bucket.SetObject(account, out)
}
