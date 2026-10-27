package ordering

import (
	"errors"
	"strings"
	"sync"
	"time"

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

// load reads one shelf's order. A key missing means an empty order.
func load(key string) (Record, error) {
	var rec Record
	if err := bucket.GetObject(key, &rec); err != nil {
		if errors.Is(err, kv.ErrKeyNotFound) || errors.Is(err, kv.ErrNotStarted) {
			return Record{}, nil
		}
		return Record{}, err
	}
	return clean(rec), nil
}

func (s *Service) List(scope string, account string) (Order, error) {
	scope = normalize(scope)
	account = normalize(account)
	if scope == "" {
		return nil, nil
	}

	rec, err := load(key(scope, account))
	if err != nil {
		return nil, err
	}
	return rec.Ids, nil
}

func (s *Service) Save(scope string, account string, ids Order) error {
	scope = normalize(scope)
	account = normalize(account)
	if scope == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rec := clean(Record{Scope: scope, Account: account, Ids: ids})
	rec.At = time.Now().UnixMilli()
	if err := bucket.SetObject(key(scope, account), rec); err != nil {
		return err
	}
	s.dirty = true
	return nil
}

// clean trims the ids, drops the empty ones and the duplicates, so a stored
// order never carries noise.
func clean(rec Record) Record {
	seen := make(map[string]struct{}, len(rec.Ids))
	out := make(Order, 0, len(rec.Ids))
	for _, id := range rec.Ids {
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
	rec.Ids = out
	return rec
}
