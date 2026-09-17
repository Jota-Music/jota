package ordering

import (
	"errors"
	"strings"
	"sync"

	"github.com/Jota-Music/jota/internal/kv"
)

var bucket = kv.UseBucket("ordering")

// Service stores the user's custom order for the library shelf. The order is
// local to Jota and scoped by the owning account, so different accounts keep
// separate orders. It only holds ids; it never edits the source playlists.
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

	var ids []string
	if err := bucket.GetObject(account, &ids); err != nil {
		if errors.Is(err, kv.ErrKeyNotFound) || errors.Is(err, kv.ErrNotStarted) {
			return nil, nil
		}
		return nil, err
	}
	return ids, nil
}

func (s *Service) Save(account string, ids []string) error {
	account = normalizeAccount(account)
	if account == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return bucket.SetObject(account, out)
}
