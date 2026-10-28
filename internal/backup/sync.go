package backup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Jota-Music/jota/internal/playlists"
)

// Options is how one pass differs from the plain reconciliation.
type Options struct {
	// Force writes the store over a document that cannot be read.
	Force bool
	// Import applies the merged document to this library. With it off the pass
	// still writes the union, so nothing another device holds is lost, but the
	// store keeps its own records.
	Import bool
	// Refresh resolves the snapshots again even when the document already
	// carries them, so an old copy does not live in the file forever.
	Refresh bool
	// Snapshots resolves the playlists the sources own. It is called only when
	// the document carries none, on a refreshing pass, or on a forced one:
	// downloading them every time would cost a network round trip per playlist
	// on each sync.
	Snapshots func() []playlists.Snapshot
}

// Stats reports what one pass read from the document and changed locally. The
// per-section counters are the records applied to this library, so a pass that
// does not import leaves all of them at zero. Bytes is the length of the
// document read, in both the folder and the Android document path, and InFile
// is the playlists it carried, against Playlists, which is what was applied.
type Stats struct {
	Bytes     int  `json:"bytes"`
	InFile    int  `json:"inFile"`
	Playlists int  `json:"playlists"`
	Follows   int  `json:"follows"`
	Saved     int  `json:"saved"`
	Order     int  `json:"order"`
	Settings  int  `json:"settings"`
	Sessions  int  `json:"sessions"`
	Modified  bool `json:"modified"`
	// Snapshots reports that the playlists the sources own were resolved on this
	// pass and now travel in the document, whatever the store did with it.
	Snapshots bool `json:"-"`
	// Document is the merged text the caller wrote, so the sync status can
	// fingerprint it for the next freshness check.
	Document string `json:"-"`
}

// Changed is how many records this pass applied, across every section.
func (s Stats) Changed() int {
	return s.Playlists + s.Follows + s.Saved + s.Order + s.Settings + s.Sessions
}

// read resolves the folder's backup: the canonical file when it is there,
// otherwise the newest *.json that parses as a backup. bytes is the length of
// the document read, and found is false when the folder holds no JSON at all, so
// an empty folder is not an error.
func read(folder string) (Document, string, int, bool, error) {
	canonical := filepath.Join(folder, BackupFile)
	data, err := os.ReadFile(canonical)
	if err == nil {
		doc, err := Parse(data)
		if err != nil {
			log.Printf("sync: read %s: %v", canonical, err)
			return Document{}, canonical, 0, false, fmt.Errorf("%s: %w", canonical, err)
		}
		return doc, canonical, len(data), true, nil
	}
	log.Printf("sync: read %s: %v", canonical, err)
	if !errors.Is(err, os.ErrNotExist) {
		return Document{}, canonical, 0, false, err
	}

	matches, err := filepath.Glob(filepath.Join(folder, "*.json"))
	if err != nil {
		return Document{}, canonical, 0, false, err
	}
	type candidate struct {
		path string
		mod  time.Time
	}
	found := make([]candidate, 0, len(matches))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		found = append(found, candidate{path: path, mod: info.ModTime()})
	}
	slices.SortFunc(found, func(a, b candidate) int {
		if c := b.mod.Compare(a.mod); c != 0 {
			return c
		}
		return strings.Compare(b.path, a.path)
	})

	if len(found) == 0 {
		return Document{Version: Version}, canonical, 0, false, nil
	}
	for _, c := range found {
		data, err := os.ReadFile(c.path)
		if err != nil {
			continue
		}
		doc, err := Parse(data)
		if err != nil {
			log.Printf("sync: read %s: %v", c.path, err)
			continue
		}
		log.Printf("sync: using backup %s", c.path)
		return doc, c.path, len(data), true, nil
	}
	return Document{}, canonical, 0, false, fmt.Errorf("no backup in %s", folder)
}

// Sync reconciles the document in folder with this library. Newer copies win;
// the merged document is written back atomically and the source-owned playlists
// are refreshed by the caller's resolver.
func (s *Service) Sync(folder string, opts Options) (Stats, error) {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return Stats{}, nil
	}

	doc, path, size, found, err := read(folder)
	switch {
	case err == nil:
	// A file that cannot be parsed is a truncated copy or a lost write, not a
	// library: the pass overwrites it with this device's document, so autosave
	// revives instead of failing forever. found stays true because the folder is
	// not empty. A version this build does not know and an I/O error on the mount
	// are not recoverable that way, and still fail the pass.
	case errors.Is(err, ErrUnreadable):
		log.Printf("sync: overwriting the unreadable %s: %v", path, err)
		doc = Document{Version: Version}
		path, size, found = filepath.Join(folder, BackupFile), 0, true
	case opts.Force:
		log.Printf("sync: overwriting %s: %v", folder, err)
		doc = Document{Version: Version}
		path, size, found = filepath.Join(folder, BackupFile), 0, false
	default:
		return Stats{}, fmt.Errorf("sync %s: %w", folder, err)
	}
	// Outside the lock: resolving a source is a network round trip.
	doc, refreshed := s.prepare(doc, opts.Snapshots, opts.Force || opts.Refresh)
	remote := len(doc.Playlists.Items)
	log.Printf("sync: %s: %d remote playlists", path, remote)

	// The store is applied first: the file is a backup, so a failed write must
	// never discard what another instance already knows.
	doc, stats, err := s.Merge(doc, opts.Import)
	if err != nil {
		return Stats{}, err
	}
	stats.Bytes = size
	stats.InFile = remote
	stats.Snapshots = refreshed
	// An empty folder with nothing to back up stays untouched: writing an empty
	// document would be imported as an empty library elsewhere.
	if !found && doc.Empty() {
		return stats, fmt.Errorf("no backup in %s", folder)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return stats, err
	}
	stats.Document = string(data)
	// The merged state always lands in the canonical file: a backup found under
	// another name is adopted instead of being rewritten in place.
	canonical := filepath.Join(folder, BackupFile)
	if path != canonical {
		if err := os.Rename(path, canonical); err != nil {
			return stats, err
		}
		path = canonical
	}
	if current, err := os.ReadFile(path); err == nil && bytes.Equal(current, data) {
		return stats, nil
	}
	stats.Modified = true
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return stats, err
	}
	return stats, os.Rename(tmp, path)
}
