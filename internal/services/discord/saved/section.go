package saved

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/reconcile"
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
	Newer   = reconcile.Newer
	Restore = reconcile.Restore
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
func (s *Service) MergeSection(part Section, mode reconcile.Mode, apply bool) (Section, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	local, err := s.index()
	if err != nil {
		return Section{}, 0, err
	}
	out := mergeSection(part, local, mode)
	// A background sync leaves us holding what the file holds, so nothing is left
	// to write, whether or not the pass applied it.
	if mode == reconcile.Newer {
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
	applied := reconcile.Diff(local.Items, out.Items, key)
	// An explicit restore is a local change and has to be pushed.
	if mode == reconcile.Restore && applied > 0 {
		s.dirty = true
	}
	return out, applied, nil
}

// mergeSection is the reconciliation rule itself: a union by id where the newer
// at wins, then a tombstone drops the entry it buries unless mode is
// reconcile.Restore. On a tie the tombstone wins, so a removal is not undone by
// the entry it deleted.
func mergeSection(remote, local Section, mode reconcile.Mode) Section {
	items, removed := reconcile.Merge(local.Items, remote.Items, key, when, local.Removed, remote.Removed, mode)
	return Section{Items: items, Removed: removed}
}

func key(entry Entry) string { return entry.Id }

func when(entry Entry) int64 { return entry.At }
