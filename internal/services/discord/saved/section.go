package saved

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"

	"github.com/Jota-Music/jota/internal/music"
)

// Entry is one saved playlist with the moment it was kept.
type Entry struct {
	Id       string   `json:"id"`
	Name     string   `json:"name,omitempty"`
	Cover    string   `json:"cover,omitempty"`
	Covers   []string `json:"covers,omitempty"`
	Subtitle string   `json:"subtitle,omitempty"`
	Owner    string   `json:"owner,omitempty"`
	At       int64    `json:"at"`
}

// Section is the saved playlists part of a backup document: the playlists kept
// in the library plus the tombstones of the ones removed, so a removal travels
// instead of being undone by the next device.
type Section struct {
	Items   []Entry          `json:"items"`
	Removed map[string]int64 `json:"removed,omitempty"`
}

// UnmarshalJSON reads the playlist list stored before entries carried a
// timestamp, giving every entry an unknown age.
func (s *Section) UnmarshalJSON(data []byte) error {
	var legacy []music.PlaylistSummary
	if err := json.Unmarshal(data, &legacy); err == nil {
		*s = Section{Items: make([]Entry, 0, len(legacy))}
		for _, playlist := range legacy {
			id := strings.TrimSpace(playlist.Id)
			if id == "" {
				continue
			}
			s.Items = append(s.Items, Entry{
				Id:       id,
				Name:     playlist.Name,
				Cover:    playlist.Cover,
				Covers:   playlist.Covers,
				Subtitle: playlist.Subtitle,
				Owner:    playlist.Owner,
			})
		}
		return nil
	}
	type body Section
	var in body
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	*s = Section(in)
	return nil
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

type mergeMode int

// Same rule as every other section: a sync takes the newer copy, an explicit
// import lets the file win.
const (
	// mergeNewer is a background sync: newer entries win and a tombstone keeps a
	// removal.
	mergeNewer mergeMode = iota
	// mergeRestore is an explicit import, which may undo a local removal.
	mergeRestore
)

// clean trims the stored ids and drops the empty ones. Callers hold no lock.
func clean(part Section) Section {
	items := make([]Entry, 0, len(part.Items))
	for _, item := range part.Items {
		id := strings.TrimSpace(item.Id)
		if id == "" {
			continue
		}
		item.Id = id
		items = append(items, item)
	}
	return Section{Items: items, Removed: part.Removed}
}

// Section reads the saved playlists into the saved part of a backup document.
func (s *Service) Section() (Section, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.index()
}

// MergeSection reconciles the store with the saved section of a document and
// answers the section both sides agree on, plus how many saved playlists
// changed. An explicit restore also undoes local removals. With apply off the
// union is answered but the store keeps its own list.
func (s *Service) MergeSection(part Section, mode mergeMode, apply bool) (Section, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	local, err := s.index()
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
	if reflect.DeepEqual(out, local) {
		return out, 0, nil
	}
	if err := s.write(out); err != nil {
		return out, 0, err
	}
	applied := diff(local, out)
	// An explicit restore is a local change and has to be pushed.
	if mode == mergeRestore && applied > 0 {
		s.dirty = true
	}
	return out, applied, nil
}

// mergeSection is the reconciliation rule itself: a union by id where the newer
// at wins, then a tombstone drops the entry it buries unless mode is
// mergeRestore. On a tie the tombstone wins, so a removal is not undone by the
// entry it deleted.
func mergeSection(remote Section, local Section, mode mergeMode) Section {
	items := make(map[string]int64, len(local.Items)+len(remote.Items))
	for _, part := range []Section{local, remote} {
		for _, item := range part.Items {
			if item.Id == "" {
				continue
			}
			if at, ok := items[item.Id]; !ok || item.At > at {
				items[item.Id] = item.At
			}
		}
	}
	removed := make(map[string]int64, len(local.Removed)+len(remote.Removed))
	for _, part := range []Section{local, remote} {
		for id, at := range part.Removed {
			if at > removed[id] {
				removed[id] = at
			}
		}
	}
	for id, at := range removed {
		itemAt, ok := items[id]
		if !ok {
			continue
		}
		if mode == mergeRestore || itemAt > at {
			delete(removed, id)
			continue
		}
		delete(items, id)
	}
	return assemble(items, local, remote, removed)
}

// assemble lists the union in a stable order, taking each entry's metadata from
// the copy that carries it.
func assemble(items map[string]int64, local, remote Section, removed map[string]int64) Section {
	out := Section{}
	owners := make(map[string]Entry, len(local.Items)+len(remote.Items))
	for _, part := range []Section{remote, local} {
		for _, item := range part.Items {
			if _, ok := owners[item.Id]; !ok || item.At > owners[item.Id].At {
				owners[item.Id] = item
			}
		}
	}
	for id, at := range items {
		item := owners[id]
		item.Id = id
		item.At = at
		out.Items = append(out.Items, item)
	}
	slices.SortFunc(out.Items, func(a, b Entry) int {
		return strings.Compare(a.Id, b.Id)
	})
	if len(removed) > 0 {
		out.Removed = removed
	}
	return out
}

// diff counts the playlists the store gains or loses when out replaces local.
func diff(local, out Section) int {
	have := make(map[string]struct{}, len(local.Items))
	for _, item := range local.Items {
		have[item.Id] = struct{}{}
	}
	changed := 0
	for _, item := range out.Items {
		if _, ok := have[item.Id]; !ok {
			changed++
		}
		delete(have, item.Id)
	}
	return changed + len(have)
}
