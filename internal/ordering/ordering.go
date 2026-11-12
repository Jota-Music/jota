package ordering

import (
	"strings"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/state"
)

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

func (s *Service) List(scope string, account string) (Order, error) {
	scope = normalize(scope)
	account = normalize(account)
	if scope == "" {
		return nil, nil
	}

	part, err := state.Read[Section](state.Order)
	if err != nil {
		return nil, err
	}
	for _, entry := range part {
		if normalize(entry.Scope) == scope && normalize(entry.Account) == account {
			return entry.Ids, nil
		}
	}
	return nil, nil
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

	part, err := state.Read[Section](state.Order)
	if err != nil {
		return err
	}
	out := make(Section, 0, len(part)+1)
	replaced := false
	for _, entry := range part {
		if normalize(entry.Scope) == scope && normalize(entry.Account) == account {
			out = append(out, Entry(rec))
			replaced = true
			continue
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, Entry(rec))
	}
	sortSection(out)
	if err := state.Write(state.Order, out); err != nil {
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
