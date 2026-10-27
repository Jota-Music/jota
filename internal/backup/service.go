package backup

import (
	"encoding/json"

	"github.com/Jota-Music/jota/internal/follows"
	"github.com/Jota-Music/jota/internal/ordering"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/services/discord/saved"
)

// Service is the backup document together with the stores it is built from: it
// writes every section this library holds and reconciles a document read
// elsewhere. A store left nil is simply not carried.
type Service struct {
	Playlists *playlists.Service
	Follows   *follows.Service
	Saved     *saved.Service
	Order     *ordering.Service
	Settings  SettingsStore
}

// Export builds the document a backup file carries: every section of this
// library, plus the playlists the sources own.
func (s *Service) Export(external []playlists.Snapshot) (string, error) {
	doc := Document{Version: Version}
	if s.Playlists != nil {
		section, err := s.Playlists.ExportSection(external)
		if err != nil {
			return "", err
		}
		doc.Playlists = section
	}
	if s.Follows != nil {
		section, err := s.Follows.Section()
		if err != nil {
			return "", err
		}
		doc.Follows = section
	}
	if s.Saved != nil {
		section, err := s.Saved.Section()
		if err != nil {
			return "", err
		}
		doc.Saved = section
	}
	if s.Order != nil {
		section, err := s.Order.Section()
		if err != nil {
			return "", err
		}
		doc.Order = section
	}
	if s.Settings != nil {
		section, err := s.Settings.Section()
		if err != nil {
			return "", err
		}
		doc.Settings = section
	}
	return marshal(doc)
}

// ImportDocument reads a document that came from elsewhere (a file the platform
// handed over, the document picker) and answers the document both sides agree
// on. The source-owned playlists are resolved only when the document carries
// none or refresh asks for them.
func (s *Service) ImportDocument(text string, snapshots func() []playlists.Snapshot, refresh, apply bool) (string, Stats, error) {
	doc, err := Parse([]byte(text))
	if err != nil {
		return "", Stats{}, err
	}
	doc, refreshed := s.prepare(doc, snapshots, refresh)
	remote := len(doc.Playlists.Items)

	merged, stats, err := s.merge(doc, false, apply)
	if err != nil {
		return "", Stats{}, err
	}
	document, err := marshal(merged)
	if err != nil {
		return "", Stats{}, err
	}
	stats.Bytes = len(text)
	stats.InFile = remote
	stats.Modified = document != text
	stats.Snapshots = refreshed
	stats.Document = document
	return document, stats, nil
}

// prepare resolves the playlists the sources own when the document carries
// none, on a refreshing pass, or on a forced one: downloading them every time
// would cost a network round trip per playlist. It reports whether it did.
func (s *Service) prepare(doc Document, snapshots func() []playlists.Snapshot, refresh bool) (Document, bool) {
	if s.Playlists == nil || snapshots == nil || (!refresh && len(doc.Playlists.External) > 0) {
		return doc, false
	}
	doc.Playlists = doc.Playlists.WithSnapshots(snapshots())
	return doc, true
}

// Restore applies a whole document to this library, undoing local deletions
// because the user asked for it, and reports what each section restored.
func (s *Service) Restore(text string) (Stats, error) {
	doc, err := Parse([]byte(text))
	if err != nil {
		return Stats{}, err
	}
	_, stats, err := s.merge(doc, true, true)
	return stats, err
}

// Merge reconciles a parsed document with this library and answers the document
// both sides agree on.
func (s *Service) Merge(doc Document, apply bool) (Document, Stats, error) {
	return s.merge(doc, false, apply)
}

// merge reconciles every section of a document with this library. With apply off
// the union is answered but no store is touched.
func (s *Service) merge(doc Document, restore, apply bool) (Document, Stats, error) {
	var stats Stats
	doc.Version = Version

	if s.Playlists != nil {
		mode := playlists.Newer
		if restore {
			mode = playlists.Restore
		}
		section, applied, err := s.Playlists.MergeSection(doc.Playlists, mode, apply)
		if err != nil {
			return Document{}, stats, err
		}
		doc.Playlists, stats.Playlists = section, applied
	}
	if s.Follows != nil {
		mode := follows.Newer
		if restore {
			mode = follows.Restore
		}
		section, applied, err := s.Follows.MergeSection(doc.Follows, mode, apply)
		if err != nil {
			return Document{}, stats, err
		}
		doc.Follows, stats.Follows = section, applied
	}
	if s.Saved != nil {
		mode := saved.Newer
		if restore {
			mode = saved.Restore
		}
		section, applied, err := s.Saved.MergeSection(doc.Saved, mode, apply)
		if err != nil {
			return Document{}, stats, err
		}
		doc.Saved, stats.Saved = section, applied
	}
	if s.Order != nil {
		mode := ordering.Newer
		if restore {
			mode = ordering.Restore
		}
		section, applied, err := s.Order.MergeSection(doc.Order, mode, apply)
		if err != nil {
			return Document{}, stats, err
		}
		doc.Order, stats.Order = section, applied
	}
	if s.Settings != nil {
		section, applied, err := mergeSettings(s.Settings, doc.Settings, restore, apply)
		if err != nil {
			return Document{}, stats, err
		}
		doc.Settings, stats.Settings = section, applied
	}
	return doc, stats, nil
}

// Pending reports whether a store changed since the last sync, so the manual
// sync is offered only when it has something to write.
func (s *Service) Pending() bool {
	if s.Playlists != nil && s.Playlists.Pending() {
		return true
	}
	if s.Follows != nil && s.Follows.Pending() {
		return true
	}
	if s.Saved != nil && s.Saved.Pending() {
		return true
	}
	if s.Order != nil && s.Order.Pending() {
		return true
	}
	if s.Settings != nil && s.Settings.Pending() {
		return true
	}
	return false
}

func marshal(doc Document) (string, error) {
	data, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
