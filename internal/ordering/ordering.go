package ordering

import (
	"errors"
	"strings"
	"sync"

	"github.com/Jota-Music/jota/internal/kv"
)

var bucket = kv.UseBucket("ordering")

// Library is the scope of the home library shelf. Its order predates scopes and
// is still stored under the bare account key, so List falls back to it.
const Library = "library"

// Service stores the user's custom order for a shelf. Orders are local to Jota
// and keyed by the shelf's scope plus, when the shelf belongs to one, the owning
// account, so different shelves and different accounts keep separate orders. It
// only holds ids; it never edits the source collection.
type Service struct {
	mu sync.Mutex
}

func New() *Service {
	return &Service{}
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// key groups orders by shelf. A shelf with no account, like the rooms shelf, is
// app-wide, so its scope alone keys it.
func key(scope, account string) string {
	if account == "" {
		return "scope:" + scope
	}
	return account + ":" + scope
}

func read(key string) (Order, error) {
	var ids Order
	if err := bucket.GetObject(key, &ids); err != nil {
		if errors.Is(err, kv.ErrKeyNotFound) || errors.Is(err, kv.ErrNotStarted) {
			return nil, nil
		}
		return nil, err
	}
	return ids, nil
}

func (s *Service) List(scope string, account string) (Order, error) {
	scope = normalize(scope)
	account = normalize(account)
	if scope == "" {
		return nil, nil
	}

	ids, err := read(key(scope, account))
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 && account != "" && scope == Library {
		return read(account)
	}
	return ids, nil
}

func (s *Service) Save(scope string, account string, ids Order) error {
	scope = normalize(scope)
	if scope == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]struct{}, len(ids))
	out := make(Order, 0, len(ids))
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
	return bucket.SetObject(key(scope, normalize(account)), out)
}
