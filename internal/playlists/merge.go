package playlists

import (
	"encoding/hex"
	"errors"
	"hash/fnv"
	"reflect"
	"slices"
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

// foreignId derives a stable local id for a playlist whose id came from a file
// and is not ours, so every instance maps it to the same record instead of
// inventing a new one on each sync.
func foreignId(id string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(id))
	return music.LocalPrefix + hex.EncodeToString(h.Sum(nil))
}

// storedId maps a file id to the record id it belongs to, leaving ours as is.
func storedId(id string) string {
	if strings.HasPrefix(id, music.LocalPrefix) {
		return id
	}
	return foreignId(id)
}

// wins reports whether the remote copy should replace the local one. A
// ModifiedAt of 0 marks a copy that predates timestamps, so its age is unknown
// and it only takes over an equally untimed copy that still has no songs.
func wins(remoteAt, localAt int64, remoteSongs, localSongs []string) bool {
	if remoteAt != localAt {
		return remoteAt > localAt
	}
	return remoteAt == 0 && len(localSongs) == 0 && len(remoteSongs) > 0
}

// normalizeRefs trims refs, so everything stored, exported and synced uses the
// references the catalog answered.
func normalizeRefs(refs []string) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref != "" {
			out = append(out, ref)
		}
	}
	return out
}

// normalizeTracks keys track metadata exactly like normalizeRefs keys refs.
func normalizeTracks(tracks map[string]music.Song) map[string]music.Song {
	if len(tracks) == 0 {
		return nil
	}
	out := make(map[string]music.Song, len(tracks))
	for ref, song := range tracks {
		out[strings.TrimSpace(ref)] = song
	}
	return out
}

// mergeTracks accumulates metadata: entries already present are kept and only
// missing refs are filled in, so no export ever discards what another copy knows.
func mergeTracks(base, extra map[string]music.Song) map[string]music.Song {
	if len(extra) == 0 {
		return base
	}
	if base == nil {
		base = make(map[string]music.Song, len(extra))
	}
	for ref, song := range extra {
		if _, ok := base[ref]; !ok {
			base[ref] = song
		}
	}
	return base
}

// mergeMode selects how far one reconciliation may go.
type mergeMode int

// Same rule as every other section: a sync takes the newer copy, an explicit
// import lets the file win.
const (
	// mergeNewer is a background sync: newer copies win and a tombstone keeps
	// a deletion.
	mergeNewer mergeMode = iota
	// mergeRestore is an explicit import, which may undo a local deletion.
	mergeRestore
)

// merge reconciles the store with a remote document and returns the document
// both sides agree on, plus how many stored playlists changed. With apply off it
// still returns the agreed document, but the store keeps its own playlists. The
// caller holds s.mu.
func (s *Service) merge(remote Section, mode mergeMode, apply bool) (Section, int, error) {
	ids, err := s.index()
	if err != nil {
		return Section{}, 0, err
	}
	local := make(map[string]exportPlaylist, len(ids))
	for _, id := range ids {
		pl, err := s.load(id)
		if err != nil {
			if errors.Is(err, kv.ErrKeyNotFound) {
				continue
			}
			return Section{}, 0, err
		}
		local[id] = s.portable(pl)
	}
	graves, err := s.tombstones()
	if err != nil {
		return Section{}, 0, err
	}

	out, graves, buried := mergeDocument(remote, local, graves, mode)
	applied := 0
	if apply {
		applied, err = s.applyMerge(out, graves, buried)
		if err != nil {
			return Section{}, applied, err
		}
	}
	// A background sync leaves us holding what the file holds, so nothing is left
	// to write. An explicit restore is a local change and has to be pushed.
	if mode == mergeNewer {
		s.dirty = false
	} else if applied > 0 || buried {
		s.dirty = true
	}
	return out, applied, nil
}

// mergeDocument is the reconciliation rule itself, with no storage access: it
// takes the remote document, the local snapshot and the tombstones and answers
// the document both sides agree on. Newer copies win by ModifiedAt, metadata is
// additive and tombstones keep a deletion from coming back unless mode is
// mergeRestore. The answer owns the snapshot's maps, so it may adopt them.
func mergeDocument(remote Section, local map[string]exportPlaylist, graves map[string]int64, mode mergeMode) (Section, map[string]int64, bool) {
	merged := make(map[string]exportPlaylist, len(local)+len(remote.Items))
	for id, p := range local {
		merged[id] = p
	}

	next := make(map[string]int64, len(graves))
	for id, at := range graves {
		next[id] = at
	}
	buried := false
	for id, at := range remote.Deleted {
		stored := storedId(id)
		if at > next[stored] {
			next[stored] = at
			buried = true
		}
	}

	for _, rp := range remote.Items {
		rp.Id = strings.TrimSpace(rp.Id)
		if rp.Id == "" {
			continue
		}
		rp.Name = strings.TrimSpace(rp.Name)
		if rp.Name == "" {
			rp.Name = defaultName
		}
		rp.Songs = normalizeRefs(rp.Songs)
		rp.Tracks = normalizeTracks(rp.Tracks)
		// Everything that reaches the merge is local: the playlists a source owns
		// are dropped before it.
		rp.Source = "local"

		stored := storedId(rp.Id)
		if at, ok := next[stored]; ok {
			if mode != mergeRestore && rp.ModifiedAt <= at {
				continue
			}
			delete(next, stored)
			buried = true
		}

		lp, ok := merged[stored]
		if !ok {
			merged[stored] = rp
			continue
		}
		// The newer copy wins for name and track list, but metadata is
		// additive: whatever one side knows and the other does not is kept.
		if wins(rp.ModifiedAt, lp.ModifiedAt, rp.Songs, lp.Songs) {
			rp.Tracks = mergeTracks(lp.Tracks, rp.Tracks)
			merged[stored] = rp
		} else {
			lp.Tracks = mergeTracks(lp.Tracks, rp.Tracks)
			lp.Id = rp.Id
			merged[stored] = lp
		}
	}

	out := Section{
		Items: make([]exportPlaylist, 0, len(merged)),
		// The source-owned playlists are not local state, but they must survive
		// a sync write: the document is also their backup.
		External: remote.External,
	}
	if len(next) > 0 {
		out.Deleted = next
	}
	for _, p := range merged {
		out.Items = append(out.Items, p)
	}
	slices.SortFunc(out.Items, func(a, b exportPlaylist) int {
		return strings.Compare(a.Id, b.Id)
	})
	return out, next, buried
}

// applyMerge writes an agreed document into the store and returns how many
// stored playlists changed. Existing metadata is kept when it is not in the
// document, so a write never discards what this instance already knew.
func (s *Service) applyMerge(out Section, graves map[string]int64, buried bool) (int, error) {
	applied := 0
	for _, p := range out.Items {
		id := storedId(p.Id)
		if s.setTrackCache != nil && len(p.Tracks) > 0 {
			for ref, song := range p.Tracks {
				s.setTrackCache(ref, song)
			}
		}
		name := strings.TrimSpace(p.Name)
		if name == "" {
			name = defaultName
		}
		refs := normalizeRefs(p.Songs)

		existing, err := s.load(id)
		if err == nil {
			changed := false
			tracks := mergeTracks(normalizeTracks(existing.Tracks), p.Tracks)
			if !reflect.DeepEqual(tracks, existing.Tracks) {
				existing.Tracks = tracks
				changed = true
			}
			if wins(p.ModifiedAt, existing.ModifiedAt, refs, existing.Songs) {
				existing.Name = name
				existing.Songs = refs
				existing.ModifiedAt = p.ModifiedAt
				changed = true
			}
			if changed {
				if err := s.save(existing); err != nil {
					return applied, err
				}
				applied++
			}
			continue
		}
		if !errors.Is(err, kv.ErrKeyNotFound) {
			return applied, err
		}
		pl := Playlist{Id: id, Name: name, Songs: refs, Tracks: p.Tracks, ModifiedAt: p.ModifiedAt}
		if err := s.save(pl); err != nil {
			return applied, err
		}
		ids, err := s.index()
		if err != nil {
			return applied, err
		}
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
		if err := s.setIndex(ids); err != nil {
			return applied, err
		}
		applied++
	}

	if buried {
		if err := bucket.SetObject(graveKey, graves); err != nil {
			return applied, err
		}
	}
	return applied, nil
}
