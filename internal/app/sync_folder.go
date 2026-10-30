package app

import (
	"errors"
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

	// The staged document: the native side copies the provider's file into
	// incoming.json and writes outgoing.json back, so the bytes never cross the
	// bridge as a string.
	syncIncoming = "incoming.json"
	syncOutgoing = "outgoing.json"
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
	// ReadStamp is the provider stamp of the copy the last pass read, not of what
	// we wrote: it says which remote state this device has already reconciled.
	ReadStamp int64 `json:"readStamp"`
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
	// Sealed says a session the document carried cannot be opened with the pass
	// this device holds, so a session is waiting for it instead of the panel
	// staying silent about it.
	Sealed bool `json:"sealed,omitempty"`
	// Problem is why the last pass failed. A pass that keeps failing between
	// intervals would otherwise be invisible until the next one.
	Problem string `json:"problem,omitempty"`
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

// recordSync is the read side of a pass: what it read and what it applied.
// written says the document is already in the file, which the folder path knows
// because Sync wrote it before answering. The platform path never knows: the page
// commits the merged copy afterwards and confirms with SyncWritten, so a commit
// that failed cannot leave a baseline that makes the next pass refuse to write.
// read is the provider stamp of the copy this pass read, 0 when the path has no
// stamp to offer: the folder path relies on MTime/FileSize instead.
func (a *App) recordSync(stats backup.Stats, fresh, written bool, read int64) {
	stale := stats.Modified && !fresh
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
	status.ReadStamp = read
	status.Problem = ""
	if written && stats.Modified {
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

// recordProblem keeps a failed pass in the status, so the panel can say why the
// library stopped being written instead of showing the previous pass as the last
// one. It is also logged: the panel only carries the reason to whoever looks at
// it, and a failure that repeats every interval must leave a trace.
func (a *App) recordProblem(err error) {
	log.Printf("sync: %v", err)
	var status SyncStatus
	_ = settings.GetObject(syncStatusKey, &status)
	status.At = time.Now().UnixMilli()
	status.Problem = err.Error()
	if setErr := settings.SetObject(syncStatusKey, status); setErr != nil {
		log.Printf("sync status: %v", setErr)
	}
}

// SyncProblem records a failure the page saw and Go did not: the document it
// merged could not be written back to the provider. Without it the panel would
// keep showing the pass before it as the last one, since nothing reached the
// file the other devices read.
func (a *App) SyncProblem(reason string) {
	if strings.TrimSpace(reason) == "" {
		return
	}
	a.recordProblem(errors.New(reason))
}

// SyncWritten is the platform confirming the merged copy reached the file, which
// is the only moment our write baseline may move: the bytes are what the other
// devices will read, and a stale flag they set has just stopped being true.
func (a *App) SyncWritten() error {
	return a.syncWrittenAt(syncStageDir())
}

// syncWrittenAt is split from SyncWritten so the file path can be exercised where
// no stage dir exists (every platform but Android).
func (a *App) syncWrittenAt(dir string) error {
	if dir == "" {
		return nil
	}
	path := filepath.Join(dir, syncOutgoing)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_ = os.Remove(path)
	var status SyncStatus
	_ = settings.GetObject(syncStatusKey, &status)
	status.At = time.Now().UnixMilli()
	status.Uri = a.SyncFolder()
	status.WriteAt = status.At
	status.Hash = stamp(string(data))
	status.Stale = false
	status.Problem = ""
	if setErr := settings.SetObject(syncStatusKey, status); setErr != nil {
		log.Printf("sync status: %v", setErr)
	}
	// The document now carries every section this device wrote, so nothing is
	// waiting to be saved any more.
	Synced()
	return nil
}

// touchSync moves At alone. A settled pass reconciled nothing, but it did run,
// so the panel's "last save" is the moment of that pass and not the last write.
func (a *App) touchSync() {
	var status SyncStatus
	_ = settings.GetObject(syncStatusKey, &status)
	status.At = time.Now().UnixMilli()
	if err := settings.SetObject(syncStatusKey, status); err != nil {
		log.Printf("sync status: %v", err)
	}
}

// SyncTouched is the page's settled pass: the provider copy is the one it already
// merged, so the pass reconciled nothing but did run.
func (a *App) SyncTouched() {
	a.touchSync()
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

// SyncReload applies the document the location holds to this library, letting
// the file win over local deletions because the user asked for it. Android has
// staged the provider's copy already; the desktop reads the folder.
func (a *App) SyncReload() (backup.Stats, error) {
	dir := syncStageDir()
	path := ""
	if dir != "" {
		path = filepath.Join(dir, syncIncoming)
	} else if folder := a.SyncFolder(); folder != "" {
		path = filepath.Join(folder, backup.BackupFile)
	}
	if path == "" {
		return backup.Stats{}, errors.New("sync: no location")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		a.recordProblem(err)
		return backup.Stats{}, err
	}
	stats, err := a.Backup.Restore(string(data))
	if err != nil {
		a.recordProblem(err)
		return backup.Stats{}, err
	}
	if dir != "" {
		_ = os.Remove(path)
	}
	if app := application.Get(); app != nil {
		app.Event.Emit("refresh:updated")
	}
	return stats, nil
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
		a.touchSync()
		return nil
	}
	started := time.Now()
	stats, err := a.Backup.Sync(folder, opts)
	if err != nil {
		a.recordProblem(err)
		return err
	}
	a.recordSync(stats, true, stats.Modified, 0)
	logPass("folder", started, stats.Bytes, len(stats.Document), stats.Modified)
	// The file holds the union of both sides, so whatever this device wrote into it
	// is in place and nothing is waiting to be saved.
	Synced()
	if stats.Modified {
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

// SyncDocument reconciles the copy the platform staged in incoming.json and
// answers whether the merge has to be written back, leaving the document both
// sides agree on in outgoing.json. lastModified is the provider's stamp for the
// copy it served, 0 when it reports none; trust is the caller asking for this
// copy to be reconciled now (a manual save or the forced overwrite), which skips
// the freshness rule.
func (a *App) SyncDocument(lastModified int64, trust bool) (bool, error) {
	dir := syncStageDir()
	if dir == "" {
		return false, nil
	}
	return a.syncDocumentAt(dir, lastModified, trust)
}

// syncDocumentAt reconciles the copy staged in dir. It is split from SyncDocument
// so the file path can be exercised where no stage dir exists (every platform
// but Android).
func (a *App) syncDocumentAt(dir string, lastModified int64, trust bool) (bool, error) {
	started := time.Now()
	incoming := filepath.Join(dir, syncIncoming)
	data, err := os.ReadFile(incoming)
	if err != nil {
		a.recordProblem(err)
		return false, err
	}
	merged, write, err := a.syncDocument(string(data), lastModified, trust)
	if err != nil {
		return false, err
	}
	if !write {
		_ = os.Remove(incoming)
		logPass("document", started, len(data), 0, false)
		return false, nil
	}
	if err := os.WriteFile(filepath.Join(dir, syncOutgoing), []byte(merged), 0o600); err != nil {
		a.recordProblem(err)
		return false, err
	}
	logPass("document", started, len(data), len(merged), true)
	return true, nil
}

// logPass leaves one line per reconciled document: how long the merge took and
// how much it moved. A pass that feels slow is then a number instead of a
// feeling, and a pass that runs on every resume shows up as one line per resume.
func logPass(where string, started time.Time, read, written int, modified bool) {
	state := "kept"
	if modified {
		state = "wrote"
	}
	log.Printf("sync: %s in %s: read %dKB, %s %dKB",
		where, time.Since(started).Round(time.Millisecond), read/1024, state, written/1024)
}

// syncDocument merges a backup document in memory and answers the document both
// sides agree on, plus whether it has to be written back.
func (a *App) syncDocument(data string, lastModified int64, trust bool) (string, bool, error) {
	fresh := trust || a.freshCopy(data, lastModified)
	var snapshotsOf func() []playlists.Snapshot
	refresh := false
	if fresh {
		snapshotsOf = a.snapshots()
		refresh = a.snapshotsDue()
	}
	merged, stats, err := a.Backup.ImportDocument(data, snapshotsOf, refresh, a.SyncImport())
	// A copy that will not parse is a truncated transfer, not a library: the pass
	// rebuilds the document from this device, exactly like the folder path does,
	// so a half-uploaded copy cannot stop autosave on either platform.
	if err != nil && errors.Is(err, backup.ErrUnreadable) {
		log.Printf("sync: recovering the document: %v", err)
		merged, stats, err = a.Backup.ImportDocument("", snapshotsOf, refresh, a.SyncImport())
		// The panel reports the size of the copy the pass read, not of the empty
		// document it merged from, or a recovered save looks like an empty file.
		stats.Bytes = len(data)
	}
	if err != nil {
		a.recordProblem(err)
		return "", false, err
	}
	a.recordSync(stats, fresh, false, lastModified)
	if !stats.Modified {
		// The copy already carried everything this device holds: there is nothing to
		// write, so nothing is waiting either.
		Synced()
	}
	return merged, fresh && merged != data, nil
}

// freshCopy is the rule the reading side applies to a copy a provider served.
// Only positive evidence of a stale copy skips the write: bytes that are not
// ours and a provider timestamp that did not move after our own last write. Our
// own bytes, or no timestamp at all, carry no such proof, and a merge is a union
// anyway, so writing is safe.
func (a *App) freshCopy(data string, lastModified int64) bool {
	status := a.SyncStatus()
	if status.Uri != a.SyncFolder() || status.WriteAt == 0 {
		return true
	}
	if stamp(data) == status.Hash {
		return true
	}
	return lastModified == 0 || lastModified > status.WriteAt
}

// SyncPending reports whether a manual sync could still change the document: our
// playlists moved since the last one, or it never ran and could not write.
func (a *App) SyncPending() bool {
	folder := a.SyncFolder()
	if folder == "" || a.Backup == nil {
		return false
	}
	status := a.SyncStatus()
	// A location never reconciled here, a write that failed, or a pass that failed
	// outright needs a sync whatever the stores say.
	if status.At == 0 || status.Uri != folder || status.Stale || status.Problem != "" {
		return true
	}
	return a.Backup.Pending()
}

// SyncSnapshotsDue reports whether the document's snapshots have aged enough to
// be downloaded again, so the page knows a pass is not a plain stat.
func (a *App) SyncSnapshotsDue() bool {
	return a.snapshotsDue()
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

// syncInBackground asks for a pass when auto-sync is enabled, without blocking
// the caller. Mutations arrive in bursts and every one of them reads the same
// stores, so the ask only arms a timer and the pass runs once the burst settled.
func (a *App) syncInBackground() {
	if !a.SyncAuto() || runtime.GOOS == "android" {
		return
	}
	a.syncSchedule()
}

// The burst of mutations is left to settle for syncDebounce, but a storm cannot
// postpone the pass past syncMaxWait counted from its first mutation.
var (
	syncDebounce = 3 * time.Second
	syncMaxWait  = 20 * time.Second
	syncTick     = time.Minute
)

// syncSchedule arms the one timer, restarting it on every ask.
func (a *App) syncSchedule() {
	a.syncTimerMu.Lock()
	defer a.syncTimerMu.Unlock()
	if a.syncTimer == nil {
		a.syncFirst = time.Now()
	} else {
		a.syncTimer.Stop()
	}
	wait := syncDebounce
	if left := time.Until(a.syncFirst.Add(syncMaxWait)); left < wait {
		wait = left
	}
	if wait < 0 {
		wait = 0
	}
	// The timer gets a generation instead of the pointer, so the callback never
	// reads the variable the arming call is still writing: wait can be zero, and
	// the timer would fire before `a.syncTimer` holds it.
	a.syncGen++
	gen := a.syncGen
	a.syncTimer = time.AfterFunc(wait, func() { a.syncFire(gen) })
}

// syncFire is the timer waking up. It only takes the burst it was armed for: a
// timer armed while this one was waking up stays, and runs instead.
func (a *App) syncFire(gen uint64) {
	a.syncTimerMu.Lock()
	if a.syncGen != gen {
		a.syncTimerMu.Unlock()
		return
	}
	a.syncTimer, a.syncFirst = nil, time.Time{}
	a.syncTimerMu.Unlock()
	a.syncQueue()
}

// syncQueue marks the pass and starts the loop that takes it.
func (a *App) syncQueue() {
	a.syncQueued.Store(true)
	if a.syncRunning.CompareAndSwap(false, true) {
		go a.syncLoop()
	}
}

// syncTicker runs the interval the settings promise. Android keeps its own, in
// the page: there the provider's file is staged by the native side.
func (a *App) syncTicker() {
	if runtime.GOOS == "android" || !a.syncTicking.CompareAndSwap(false, true) {
		return
	}
	go func() {
		ticker := time.NewTicker(syncTick)
		defer ticker.Stop()
		for range ticker.C {
			if a.SyncAuto() && a.SyncFolder() != "" && a.syncDue() {
				a.syncSchedule()
			}
		}
	}()
}

// syncDue reports whether the interval has passed since the last pass, or work
// is waiting for one. A location never reconciled counts as due.
func (a *App) syncDue() bool {
	if a.SyncPending() {
		return true
	}
	return time.Since(time.UnixMilli(a.SyncStatus().At)) >= time.Duration(a.SyncInterval())*time.Minute
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
