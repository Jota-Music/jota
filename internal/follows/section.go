package follows

import (
	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/reconcile"
	"github.com/Jota-Music/jota/internal/state"
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
	Newer   = reconcile.Newer
	Restore = reconcile.Restore
)

// clean trims the stored ids, drops the empty ones and names every follow the
// way the app names ids, so a list or a document written before the rule is
// answered in the current form. Callers hold no lock.
func clean(list Section) Section {
	items := make([]Entry, 0, len(list.Items))
	for _, item := range list.Items {
		id := music.StoredFollowID(item.Id)
		if id == "" {
			continue
		}
		items = append(items, Entry{Id: id, At: item.At})
	}

	removed := make(map[string]int64, len(list.Removed))
	for id, at := range list.Removed {
		key := music.StoredFollowID(id)
		if key == "" {
			continue
		}
		if at > removed[key] {
			removed[key] = at
		}
	}
	return Section{Items: items, Removed: removed}
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
func (s *Service) MergeSection(part Section, mode reconcile.Mode, apply bool) (Section, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	local, err := s.load()
	if err != nil {
		return Section{}, 0, err
	}
	// The file may name a follow the way it was written before the rule, and
	// the merge keys on the id: both sides have to speak the same form.
	out := mergeSection(clean(part), local, mode)
	// A background sync leaves us holding what the file holds, so nothing is left
	// to write, whether or not the pass applied it.
	if mode == reconcile.Newer {
		s.dirty = false
	}
	if !apply {
		return out, 0, nil
	}

	applied := reconcile.Diff(local.Items, out.Items, key)
	if applied > 0 {
		if err := state.Write(state.Follows, out); err != nil {
			return out, applied, err
		}
	}
	// An explicit restore is a local change and has to be pushed.
	if mode == reconcile.Restore && applied > 0 {
		s.dirty = true
	}
	return out, applied, nil
}

// mergeSection is the reconciliation rule itself: a union by id where the newer
// at wins, then a tombstone drops the entry it buries unless mode is
// reconcile.Restore. On a tie the tombstone wins, so an unfollow is not undone
// by the entry it deleted.
func mergeSection(remote, local Section, mode reconcile.Mode) Section {
	items, removed := reconcile.Merge(local.Items, remote.Items, key, when, local.Removed, remote.Removed, mode)
	return Section{Items: items, Removed: removed}
}

func key(entry Entry) string { return entry.Id }

func when(entry Entry) int64 { return entry.At }
