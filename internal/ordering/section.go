package ordering

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"

	"github.com/Jota-Music/jota/internal/kv"
)

// Record is one shelf's order: the ids in sequence, the shelf's scope and
// owner, and the moment the shelf was last saved, so the newest order travels
// as a whole.
type Record struct {
	Scope   string   `json:"scope,omitempty"`
	Account string   `json:"account,omitempty"`
	Ids     []string `json:"ids"`
	At      int64    `json:"at"`
}

// Entry is one shelf's order inside a backup document. The scope names the
// shelf and the account names the owner when the shelf belongs to one.
type Entry struct {
	Scope   string   `json:"scope"`
	Account string   `json:"account,omitempty"`
	Ids     []string `json:"ids"`
	At      int64    `json:"at"`
}

// Section is the ordering part of a backup document: every stored order, one
// entry per shelf, sorted by scope and account so the file stays stable.
type Section []Entry

// Empty reports whether the section carries nothing worth writing.
func (s Section) Empty() bool {
	return len(s) == 0
}

// shelf identifies one order: a scope alone when the shelf is app-wide, the
// scope plus the owning account otherwise.
type shelf struct {
	scope   string
	account string
}

func (e Entry) shelf() shelf {
	return shelf{normalize(e.Scope), normalize(e.Account)}
}

func (r Record) shelf() shelf {
	return shelf{normalize(r.Scope), normalize(r.Account)}
}

// Merge modes, for the callers outside this package that decide between a
// background sync and an explicit restore.
const (
	Newer   = mergeNewer
	Restore = mergeRestore
)

type mergeMode int

// Same rule as every other section: a sync takes the newer copy, an explicit
// import lets the file win.
const (
	// mergeNewer is a background sync: the newest list wins.
	mergeNewer mergeMode = iota
	// mergeRestore is an explicit import, where the file's list always wins.
	mergeRestore
)

// Section reads every stored order into the ordering part of a backup document.
func (s *Service) Section() (Section, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.section()
}

// section reads every stored order. Callers must hold mu.
func (s *Service) section() (Section, error) {
	out := make(Section, 0)
	err := bucket.ForEach(func(_ string, data []byte) error {
		var rec Record
		if err := json.Unmarshal(data, &rec); err != nil {
			return nil
		}
		rec.Scope, rec.Account = normalize(rec.Scope), normalize(rec.Account)
		if rec.Scope == "" {
			return nil
		}
		out = append(out, Entry(clean(rec)))
		return nil
	})
	if errors.Is(err, kv.ErrNotStarted) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	sortSection(out)
	return out, nil
}

// sortSection orders the entries by shelf, so the same store always writes the
// same bytes and a pass does not rewrite the file over a reordering.
func sortSection(part Section) {
	sort.Slice(part, func(i, j int) bool {
		if part[i].Scope != part[j].Scope {
			return part[i].Scope < part[j].Scope
		}
		return part[i].Account < part[j].Account
	})
}

// index groups a section by shelf, so both sides reconcile on the same pair.
func index(part Section) map[shelf]Record {
	out := make(map[shelf]Record, len(part))
	for _, entry := range part {
		out[entry.shelf()] = Record(entry)
	}
	return out
}

// MergeSection reconciles the store with the ordering section of a document and
// answers the section both sides agree on, plus how many shelves changed. A whole
// list wins by its timestamp, because merging positions is meaningless; an
// explicit restore lets the file's list win even when the local one is newer.
// With apply off the union is answered but the store keeps its own orders.
func (s *Service) MergeSection(part Section, mode mergeMode, apply bool) (Section, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	local, err := s.section()
	if err != nil {
		return nil, 0, err
	}
	out := mergeSection(part, local, mode)
	// A background sync leaves us holding what the file holds, so nothing is left
	// to write, whether or not the pass applied it.
	if mode == mergeNewer {
		s.dirty = false
	}
	if !apply {
		return out, 0, nil
	}

	held := index(local)
	applied := 0
	for _, entry := range out {
		rec := Record(entry)
		if cur, ok := held[entry.shelf()]; ok && reflect.DeepEqual(cur, rec) {
			continue
		}
		if err := bucket.SetObject(key(entry.Scope, entry.Account), rec); err != nil {
			return out, applied, err
		}
		applied++
	}
	// An explicit restore is a local change and has to be pushed.
	if mode == mergeRestore && applied > 0 {
		s.dirty = true
	}
	return out, applied, nil
}

// mergeSection is the reconciliation rule itself: every shelf keeps the list of
// the copy that saved last, with no per-id merge.
func mergeSection(remote Section, local Section, mode mergeMode) Section {
	shelves := index(local)
	for _, entry := range remote {
		entry.Scope = normalize(entry.Scope)
		entry.Account = normalize(entry.Account)
		if entry.Scope == "" {
			continue
		}
		rec := clean(Record(entry))
		if cur, ok := shelves[rec.shelf()]; ok && mode != mergeRestore && cur.At >= rec.At {
			continue
		}
		shelves[rec.shelf()] = rec
	}
	if len(shelves) == 0 {
		return nil
	}
	out := make(Section, 0, len(shelves))
	for _, rec := range shelves {
		out = append(out, Entry(rec))
	}
	sortSection(out)
	return out
}
