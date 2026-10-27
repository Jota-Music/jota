package follows

import (
	"slices"
	"strings"
)

// Entry is one followed user with the moment it was followed.
type Entry struct {
	Id string `json:"id"`
	At int64  `json:"at"`
}

// Section is the follows part of a backup document: the followed ids plus the
// tombstones of the ones unfollowed, so a removal travels instead of being
// undone by the next device.
type Section struct {
	Items   []Entry          `json:"items,omitempty"`
	Removed map[string]int64 `json:"removed,omitempty"`
}

// Empty reports whether the section carries nothing worth writing.
func (s Section) Empty() bool {
	return len(s.Items) == 0 && len(s.Removed) == 0
}

// Merge modes, for the callers outside this package that decide between a
// background sync and an explicit restore.
const (
	Newer   = mergeNewer
	Restore = mergeRestore
)

// Same rule as every other section: a sync takes the newer copy, an explicit
// import lets the file win.
type mergeMode int

const (
	// mergeNewer is a background sync: newer entries win and a tombstone keeps
	// an unfollow.
	mergeNewer mergeMode = iota
	// mergeRestore is an explicit import, which may undo a local unfollow.
	mergeRestore
)

// clean trims the stored ids and drops the empty ones. Callers hold no lock.
func clean(list Section) Section {
	items := make([]Entry, 0, len(list.Items))
	for _, item := range list.Items {
		id := strings.TrimSpace(item.Id)
		if id == "" {
			continue
		}
		items = append(items, Entry{Id: id, At: item.At})
	}
	return Section{Items: items, Removed: list.Removed}
}

// Section reads the stored list into the follows part of a backup document.
func (s *Service) Section() (Section, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.load()
}

// MergeSection reconciles the store with the follows section of a document and
// answers the section both sides agree on, plus how many follows changed. An
// explicit restore also undoes local unfollows. With apply off the union is
// answered but the store keeps its own follows.
func (s *Service) MergeSection(part Section, mode mergeMode, apply bool) (Section, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	local, err := s.load()
	if err != nil {
		return Section{}, 0, err
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

	applied := diff(local, out)
	if applied > 0 {
		if err := bucket.SetObject(listKey, out); err != nil {
			return out, applied, err
		}
	}
	// An explicit restore is a local change and has to be pushed.
	if mode == mergeRestore && applied > 0 {
		s.dirty = true
	}
	return out, applied, nil
}

// mergeSection is the reconciliation rule itself: a union by id where the newer
// at wins, then a tombstone drops the entry it buries unless mode is
// mergeRestore. On a tie the tombstone wins, so an unfollow is not undone by the
// entry it deleted.
func mergeSection(remote, local Section, mode mergeMode) Section {
	items := make(map[string]int64, len(local.Items)+len(remote.Items))
	for _, list := range []Section{local, remote} {
		for _, item := range list.Items {
			if item.Id == "" {
				continue
			}
			if at, ok := items[item.Id]; !ok || item.At > at {
				items[item.Id] = item.At
			}
		}
	}
	removed := make(map[string]int64, len(local.Removed)+len(remote.Removed))
	for _, graves := range []map[string]int64{local.Removed, remote.Removed} {
		for id, at := range graves {
			if at > removed[id] {
				removed[id] = at
			}
		}
	}
	for id, at := range removed {
		item, ok := items[id]
		if !ok {
			continue
		}
		if mode == mergeRestore || item > at {
			delete(removed, id)
			continue
		}
		delete(items, id)
	}
	return Section{Items: entries(items), Removed: graves(removed)}
}

// entries lists the union in a stable order, so the document does not churn.
func entries(items map[string]int64) []Entry {
	if len(items) == 0 {
		return nil
	}
	out := make([]Entry, 0, len(items))
	for id, at := range items {
		out = append(out, Entry{Id: id, At: at})
	}
	slices.SortFunc(out, func(a, b Entry) int {
		return strings.Compare(a.Id, b.Id)
	})
	return out
}

func graves(removed map[string]int64) map[string]int64 {
	if len(removed) == 0 {
		return nil
	}
	return removed
}

// diff counts the users the store gains or loses when list replaces local.
func diff(local, list Section) int {
	have := make(map[string]struct{}, len(local.Items))
	for _, item := range local.Items {
		have[item.Id] = struct{}{}
	}
	changed := 0
	for _, item := range list.Items {
		if _, ok := have[item.Id]; !ok {
			changed++
		}
		delete(have, item.Id)
	}
	return changed + len(have)
}
