// Package reconcile is the rule every section of the backup document follows:
// timestamped records reconcile by id, and a tombstone keeps a removal from
// coming back on the next pass.
package reconcile

import (
	"slices"
	"strings"
)

// Mode selects how far one reconciliation may go.
type Mode int

const (
	// Newer is a background sync: the newest copy wins and a tombstone keeps a
	// removal.
	Newer Mode = iota
	// Restore is an explicit import, which may undo a local removal.
	Restore
)

// Merge reconciles two lists of records and applies the tombstones both sides
// hold, answering the union plus the tombstones left standing. The newer copy
// of an id wins, on a tie the tombstone does, so a removal is not undone by the
// record it deleted. Restore lets the document win outright. The union is
// sorted by id, so the document does not churn.
func Merge[T any](
	local, remote []T,
	id func(T) string,
	at func(T) int64,
	localRemoved, remoteRemoved map[string]int64,
	mode Mode,
) ([]T, map[string]int64) {
	// The document's copy is seen first, so a tie leaves its metadata in place
	// and only a strictly newer local copy replaces it.
	items := make(map[string]T, len(local)+len(remote))
	ages := make(map[string]int64, len(local)+len(remote))
	for _, list := range [][]T{remote, local} {
		for _, item := range list {
			key := id(item)
			if key == "" {
				continue
			}
			if age, ok := ages[key]; !ok || at(item) > age {
				items[key] = item
				ages[key] = at(item)
			}
		}
	}

	removed := make(map[string]int64, len(localRemoved)+len(remoteRemoved))
	for _, graves := range []map[string]int64{localRemoved, remoteRemoved} {
		for key, age := range graves {
			if age > removed[key] {
				removed[key] = age
			}
		}
	}
	for key, age := range removed {
		held, ok := ages[key]
		if !ok {
			continue
		}
		if mode == Restore || held > age {
			delete(removed, key)
			continue
		}
		delete(items, key)
	}

	var out []T
	if len(items) > 0 {
		out = make([]T, 0, len(items))
		for _, item := range items {
			out = append(out, item)
		}
		slices.SortFunc(out, func(a, b T) int {
			return strings.Compare(id(a), id(b))
		})
	}
	if len(removed) == 0 {
		removed = nil
	}
	return out, removed
}

// Diff counts the records the store gains or loses when out replaces local.
func Diff[T any](local, out []T, id func(T) string) int {
	have := make(map[string]struct{}, len(local))
	for _, item := range local {
		have[id(item)] = struct{}{}
	}
	changed := 0
	for _, item := range out {
		key := id(item)
		if _, ok := have[key]; !ok {
			changed++
		}
		delete(have, key)
	}
	return changed + len(have)
}
