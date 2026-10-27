// Package backup owns the file the library travels in: one JSON document with
// every section a device needs to rebuild itself, and the reconciliation that
// keeps the copies converging.
package backup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/Jota-Music/jota/internal/follows"
	"github.com/Jota-Music/jota/internal/ordering"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/services/discord/saved"
)

// Version is the only document format read and written.
const Version = 1

// BackupFile is the name Export and Sync share, so the exported file is always
// recognized as the sync document.
const BackupFile = "jota.backup.json"

// The document is written compact: on a real 4.6 MB library, indentation was 40%
// of the bytes, and the file is synced, not read. Everything that matters for
// its size lives in the sections, which share metadata (size_test.go pins that);
// gzip was measured at 4.4x and left out because it would need a binary bridge
// on Android while the providers already compress what they store.

// Document is the backup file. Each section belongs to the package that owns
// its records, so the document only lays them out side by side.
type Document struct {
	Version   int
	Playlists playlists.Section
	Follows   follows.Section
	Saved     saved.Section
	Order     ordering.Section
	Settings  Settings
	Sessions  Sessions
}

// Empty reports whether the document carries no section at all.
func (d Document) Empty() bool {
	return d.Playlists.Empty() &&
		d.Follows.Empty() &&
		d.Saved.Empty() &&
		d.Order.Empty() &&
		len(d.Settings) == 0 &&
		len(d.Sessions) == 0
}

// MarshalJSON omits the sections the library has nothing for, so an empty
// section never occupies the file.
func (d Document) MarshalJSON() ([]byte, error) {
	type body struct {
		Version   int                `json:"version"`
		Playlists *playlists.Section `json:"playlists,omitempty"`
		Follows   *follows.Section   `json:"follows,omitempty"`
		Saved     *saved.Section     `json:"saved,omitempty"`
		Order     *ordering.Section  `json:"order,omitempty"`
		Settings  *Settings          `json:"settings,omitempty"`
		Sessions  *Sessions          `json:"sessions,omitempty"`
	}
	out := body{Version: d.Version}
	if !d.Playlists.Empty() {
		out.Playlists = &d.Playlists
	}
	if !d.Follows.Empty() {
		out.Follows = &d.Follows
	}
	if !d.Saved.Empty() {
		out.Saved = &d.Saved
	}
	if !d.Order.Empty() {
		out.Order = &d.Order
	}
	if len(d.Settings) > 0 {
		out.Settings = &d.Settings
	}
	if len(d.Sessions) > 0 {
		out.Sessions = &d.Sessions
	}
	return json.Marshal(out)
}

// UnmarshalJSON accepts only a document of the version this build writes.
func (d *Document) UnmarshalJSON(data []byte) error {
	type body Document
	var in body
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	if in.Version != Version {
		return fmt.Errorf("%w: version %d", ErrUnreadable, in.Version)
	}
	*d = Document(in)
	return nil
}

// ErrUnreadable is the one failure a caller can act on: a file this build
// cannot read, which the user solves by overwriting it. The detail goes to the
// log, never to the panel, because a JSON decoder message says nothing useful.
var ErrUnreadable = errors.New("the backup file was written by another version")

// Parse reads a document. A blank file is an empty one, so a backup file that
// was just created can be filled in instead of being reported as broken.
func Parse(data []byte) (Document, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return Document{Version: Version}, nil
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		log.Printf("backup: %v", err)
		return Document{}, ErrUnreadable
	}
	return doc, nil
}
