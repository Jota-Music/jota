package app

import (
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Jota-Music/jota/internal/backup"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/refresh"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	syncFolderKey = "sync-folder"
	syncStatusKey = "sync-status"
)

// SyncStatus is what the last sync read and changed. It lives in settings
// because the WebView loses its in-memory state on every reload, and the panel
// still has to say what happened.
type SyncStatus struct {
	At  int64  `json:"at"`
	Uri string `json:"uri"`
	// WriteAt is when we last actually wrote the document, and Hash fingerprints
	// what we wrote. A read that only pulled must not move them: they are the
	// baseline the freshness check compares against.
	WriteAt int64  `json:"writeAt"`
	Hash    uint32 `json:"hash"`
	// SnapshotsAt is when the playlists the sources own were last written into
	// the document. Their download is the expensive part of a pass, so they are
	// only resolved again once they are older than the refresh interval.
	SnapshotsAt int64 `json:"snapshotsAt"`
	// MTime and FileSize fingerprint the canonical document this device last
	// reconciled, so an unchanged folder is skipped without reading it.
	MTime    int64 `json:"mtime"`
	FileSize int64 `json:"fileSize"`
	Bytes    int   `json:"bytes"`
	Remote   int   `json:"remote"`
	Imported int   `json:"imported"`
	Modified bool  `json:"modified"`
	Stale    bool  `json:"stale"`
}

func (a *App) SyncStatus() SyncStatus {
	var status SyncStatus
	if err := settings.GetObject(syncStatusKey, &status); err != nil {
		return SyncStatus{}
	}
	return status
}

// stamp fingerprints the document we wrote, so a provider that reports no
// timestamp is still judged against our own copy after a WebView reload.
func stamp(document string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(document))
	return h.Sum32()
}

func (a *App) recordSync(stats backup.Stats, fresh bool) {
	stale := stats.Modified && !fresh
	wrote := stats.Modified && fresh
	var uri string
	_ = settings.GetObject(syncFolderKey, &uri)
	var status SyncStatus
	_ = settings.GetObject(syncStatusKey, &status)
	status.At = time.Now().UnixMilli()
	status.Uri = uri
	status.Bytes = stats.Bytes
	status.Remote = stats.InFile
	status.Imported = stats.Changed()
	status.Modified = stats.Modified
	status.Stale = stale
	if wrote {
		status.WriteAt = status.At
		status.Hash = stamp(stats.Document)
	}
	// A stale read never wrote the document, so the copy in the file is not the
	// snapshots this pass resolved.
	if stats.Snapshots && !stale {
		status.SnapshotsAt = status.At
	}
	status.MTime, status.FileSize = 0, 0
	if uri != "" {
		if info, err := os.Stat(filepath.Join(uri, backup.BackupFile)); err == nil {
			status.MTime = info.ModTime().UnixNano()
			status.FileSize = info.Size()
		}
	}
	if err := settings.SetObject(syncStatusKey, status); err != nil {
		log.Printf("sync status: %v", err)
	}
}
func (a *App) SyncFolder() string {
	var folder string
	if err := settings.GetObject(syncFolderKey, &folder); err != nil {
		return ""
	}
	return folder
}

func (a *App) SetSyncFolder(folder string) error {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return settings.SetObject(syncFolderKey, "")
	}
	// Android keeps a content:// document URI here, not a filesystem path.
	if runtime.GOOS != "android" {
		if info, err := os.Stat(folder); err != nil || !info.IsDir() {
			return err
		}
	}
	return settings.SetObject(syncFolderKey, folder)
}

func (a *App) ChooseSyncFolder() (string, error) {
	path, err := application.Get().Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	if err := a.SetSyncFolder(path); err != nil {
		return "", err
	}
	return path, nil
}

// SyncAuto decides whether focus, the app lifecycle and the interval reconcile
// the folder. Missing means on, like SyncImport: the switch is new and the
// document already travels without it.
func (a *App) SyncAuto() bool {
	v := readSyncSettings()
	if v.Auto == nil {
		return true
	}
	return *v.Auto
}

func (a *App) SetSyncAuto(enabled bool) error {
	v := readSyncSettings()
	v.Auto = &enabled
	if err := settings.SetObject("sync", v); err != nil {
		return err
	}
	touch("sync")
	a.syncInBackground()
	return nil
}

type syncSettings struct {
	Import   *bool `json:"import,omitempty"`
	Interval *int  `json:"interval,omitempty"`
	Auto     *bool `json:"auto,omitempty"`
}

// readSyncSettings reads the compound key. An install upgraded from the split
// keys still holds sync-import and sync-interval, so they are read as a
// fallback until the next write moves them into the compound.
func readSyncSettings() syncSettings {
	var v syncSettings
	_ = settings.GetObject("sync", &v)
	if v.Import == nil {
		var imported bool
		if err := settings.GetObject("sync-import", &imported); err == nil {
			v.Import = &imported
		}
	}
	if v.Interval == nil {
		var interval int
		if err := settings.GetObject("sync-interval", &interval); err == nil {
			v.Interval = &interval
		}
	}
	if v.Auto == nil {
		var auto bool
		if err := settings.GetObject("sync-auto", &auto); err == nil {
			v.Auto = &auto
		}
	}
	return v
}

// SyncImport decides whether a pass also applies the document to this library.
// Missing means on: that was the only behaviour before the switch existed.
func (a *App) SyncImport() bool {
	v := readSyncSettings()
	if v.Import == nil {
		return true
	}
	return *v.Import
}

func (a *App) SetSyncImport(enabled bool) error {
	v := readSyncSettings()
	v.Import = &enabled
	if err := settings.SetObject("sync", v); err != nil {
		return err
	}
	touch("sync")
	a.syncInBackground()
	return nil
}

func (a *App) SyncInterval() int {
	v := readSyncSettings()
	if v.Interval == nil || *v.Interval < 1 {
		return 15
	}
	return *v.Interval
}

func (a *App) SetSyncInterval(minutes int) error {
	if minutes < 1 {
		minutes = 15
	}
	v := readSyncSettings()
	v.Interval = &minutes
	if err := settings.SetObject("sync", v); err != nil {
		return err
	}
	touch("sync")
	a.syncInBackground()
	return nil
}

// SyncPlaylists is the automatic pass: it reconciles the document and only
// downloads the playlists the sources own once their snapshot is old.
func (a *App) SyncPlaylists() error {
	return a.syncFolder(a.options(false, a.snapshotsDue()), true)
}

// SyncSave is the manual save: it writes this library now, and only resolves the
// playlists the sources own once their snapshot is old, exactly like the
// automatic pass. Downloading the whole account on every click is what made the
// button look frozen.
func (a *App) SyncSave() error {
	return a.syncFolder(a.options(false, a.snapshotsDue()), false)
}

// SyncOverwrite is the emergency exit: it writes our document over a file a
// normal save cannot read.
func (a *App) SyncOverwrite() error {
	return a.syncFolder(a.options(true, false), false)
}

func (a *App) options(force, refresh bool) backup.Options {
	return backup.Options{
		Force:     force,
		Import:    a.SyncImport(),
		Refresh:   refresh,
		Snapshots: a.snapshots(),
	}
}

// settle lets an unchanged folder answer an automatic pass with a stat instead
// of a full read and merge. A save the user asked for always runs: skipping it
// would leave the panel showing the previous pass as the last one.
func (a *App) syncFolder(opts backup.Options, settle bool) error {
	folder := a.SyncFolder()
	if folder == "" {
		return nil
	}
	// Two passes at once would write the same temp file over each other, and the
	// triggers now arrive from many callers at once.
	a.syncMu.Lock()
	defer a.syncMu.Unlock()
	if settle && !opts.Refresh && a.syncSettled(folder) {
		return nil
	}
	stats, err := a.Backup.Sync(folder, opts)
	if err != nil {
		return err
	}
	a.recordSync(stats, true)
	if stats.Modified {
		Synced()
		if app := application.Get(); app != nil {
			app.Event.Emit("refresh:updated")
		}
	}
	return nil
}

// syncSettled reports whether the folder still holds the copy this device last
// reconciled and the library has nothing new to write, which makes a pass a
// stat call instead of a full read and merge.
func (a *App) syncSettled(folder string) bool {
	if a.SyncPending() {
		return false
	}
	info, err := os.Stat(filepath.Join(folder, backup.BackupFile))
	if err != nil {
		return false
	}
	status := a.SyncStatus()
	return info.ModTime().UnixNano() == status.MTime && info.Size() == status.FileSize
}

// SyncDocument merges a backup document read from the platform picker (an
// Android content:// document) and returns the document both sides agree on.
// fresh reports whether the copy is newer than our own last write: only the
// side that read the file can tell a cache miss from a cached copy.
func (a *App) SyncDocument(data string, fresh bool) (string, error) {
	var snapshotsOf func() []playlists.Snapshot
	refresh := false
	if fresh {
		snapshotsOf = a.snapshots()
		refresh = a.snapshotsDue()
	}
	merged, stats, err := a.Backup.ImportDocument(data, snapshotsOf, refresh, a.SyncImport())
	if err != nil {
		return "", err
	}
	a.recordSync(stats, fresh)
	if stats.Modified && fresh {
		Synced()
	}
	return merged, nil
}

// SyncPending reports whether a manual sync could still change the document: our
// playlists moved since the last one, or it never ran and could not write.
func (a *App) SyncPending() bool {
	folder := a.SyncFolder()
	if folder == "" || a.Backup == nil {
		return false
	}
	status := a.SyncStatus()
	// A location never reconciled here, or a write that failed, needs a sync
	// whatever the stores say.
	if status.At == 0 || status.Uri != folder || status.Stale {
		return true
	}
	return a.Backup.Pending()
}

// snapshots is the source resolution the sync passes to the service. It is lazy
// on purpose: the backup carries every playlist, but a document that already has
// them is not downloaded again on each pass.
func (a *App) snapshots() func() []playlists.Snapshot {
	return func() []playlists.Snapshot {
		return a.externalPlaylists(a.externalSummaries(), false)
	}
}

// snapshotsDue reports whether the snapshot the document carries has aged past
// the same interval the rest of the cached content is refreshed at.
func (a *App) snapshotsDue() bool {
	at := a.SyncStatus().SnapshotsAt
	return at == 0 || time.Since(time.UnixMilli(at)) >= refresh.Interval
}

// syncInBackground triggers a sync when auto-sync is enabled, without blocking
// the caller.
func (a *App) syncInBackground() {
	if !a.SyncAuto() || runtime.GOOS == "android" {
		return
	}
	a.syncQueued.Store(true)
	if a.syncRunning.CompareAndSwap(false, true) {
		go a.syncLoop()
	}
}

func (a *App) syncLoop() {
	for {
		for a.syncQueued.Swap(false) {
			if err := a.SyncPlaylists(); err != nil {
				log.Printf("sync: %v", err)
			}
		}
		a.syncRunning.Store(false)
		// A trigger that landed after the last swap would be lost, so the clear
		// is followed by a re-check: either this round takes it or the caller
		// that won the flag does.
		if !a.syncQueued.Load() || !a.syncRunning.CompareAndSwap(false, true) {
			return
		}
	}
}
