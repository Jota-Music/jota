package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/backup"
	"github.com/Jota-Music/jota/internal/follows"
	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/ordering"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/services/discord/saved"
	"github.com/Jota-Music/jota/internal/services/spotify"
	"github.com/Jota-Music/jota/internal/services/youtube"
)

func syncApp(t *testing.T) *App {
	t.Helper()
	// The settings store is a package-level value: a mark left dirty by another
	// test would leak into this one.
	settingsSync.dirty.Store(false)
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)
	// The backup also carries the playlists of the sources, so the app needs them
	// wired even when none of them is signed in.
	lists, list, order, kept := playlists.New(nil), follows.New(), ordering.New(), saved.New()
	return &App{
		Playlists: lists,
		Spotify:   spotify.NewSpotifyService(""),
		YouTube:   youtube.NewService(),
		Follows:   list,
		Order:     order,
		Saved:     kept,
		Backup: &backup.Service{
			Playlists: lists,
			Follows:   list,
			Saved:     kept,
			Order:     order,
			Settings:  settingsSync,
			Sessions:  sessionsSync,
		},
	}
}

// The panel hides the manual sync once its last pass answered everything, so
// the three states that still need it must keep reporting true.
func TestSyncPending(t *testing.T) {
	a := syncApp(t)
	if a.SyncPending() {
		t.Fatal("no sync folder reported a pending sync")
	}

	folder := t.TempDir()
	if err := a.SetSyncFolder(folder); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if !a.SyncPending() {
		t.Fatal("a never synced folder reported nothing to sync")
	}

	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if a.SyncPending() {
		t.Fatal("a synced folder reported a pending sync")
	}

	if _, err := a.Playlists.Create("B Sides"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	if !a.SyncPending() {
		t.Fatal("a local change after the last sync was not reported")
	}

	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if _, err := a.Backup.Restore(`{"version":1,"playlists":{"items":[{"id":"x","source":"local","name":"Imported"}]}}`); err != nil {
		t.Fatalf("import: %v", err)
	}
	if !a.SyncPending() {
		t.Fatal("an explicit restore was not reported as pending")
	}
}

// A focus pass that lands on the same canonical file and has nothing local to
// write costs a stat, not a full read and merge.
func TestSyncSkipsUnchangedFolder(t *testing.T) {
	a := syncApp(t)
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	counted := &passes{}
	a.Backup.Settings = counted
	before := a.SyncStatus().At
	time.Sleep(2 * time.Millisecond)
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if got := counted.count.Load(); got != 0 {
		t.Fatalf("unchanged folder ran %d passes, want 0", got)
	}
	// The skipped pass still reconciled, so the panel must not keep showing the
	// last write as the last save.
	if after := a.SyncStatus().At; after <= before {
		t.Fatalf("settled pass left At at %d, want later than %d", after, before)
	}

	if _, err := a.Playlists.Create("B Sides"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if got := counted.count.Load(); got != 1 {
		t.Fatalf("a local change ran %d passes, want 1", got)
	}
}

// The save button is an explicit ask: an unchanged folder still runs a pass and
// records it, or the panel keeps the previous pass as the last save.
func TestSyncSaveRunsOnUnchangedFolder(t *testing.T) {
	a := syncApp(t)
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	before := a.SyncStatus().At

	counted := &passes{}
	a.Backup.Settings = counted
	// At is a millisecond stamp, so the save has to land in a later one.
	time.Sleep(2 * time.Millisecond)
	if err := a.SyncSave(); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := counted.count.Load(); got != 1 {
		t.Fatalf("a manual save ran %d passes, want 1", got)
	}
	if after := a.SyncStatus().At; after <= before {
		t.Fatalf("manual save left At at %d, want later than %d", after, before)
	}
}

// A pass that fails leaves the reason in the status, and the next good pass
// clears it, or a broken folder stays silent until someone reads the log.
func TestSyncRecordsFailure(t *testing.T) {
	a := syncApp(t)
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	a.Backup.Settings = broken{}
	if err := a.SyncSave(); err == nil {
		t.Fatal("a section that cannot be read did not fail the pass")
	}
	if problem := a.SyncStatus().Problem; problem == "" {
		t.Fatal("a failed pass left no problem in the status")
	}

	a.Backup.Settings = settingsSync
	if err := a.SyncSave(); err != nil {
		t.Fatalf("save: %v", err)
	}
	if problem := a.SyncStatus().Problem; problem != "" {
		t.Fatalf("a good pass kept the problem %q", problem)
	}
}

// A due snapshot refresh is work a pass still has to do even when the file did
// not move.
func TestSyncRefreshesDueSnapshots(t *testing.T) {
	a := syncApp(t)
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	counted := &passes{}
	a.Backup.Settings = counted
	status := a.SyncStatus()
	status.SnapshotsAt = 0
	if err := settings.SetObject(syncStatusKey, status); err != nil {
		t.Fatalf("status: %v", err)
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if got := counted.count.Load(); got != 1 {
		t.Fatalf("due snapshots ran %d passes, want 1", got)
	}
}

// The panel's reload applies the document the location holds, the file winning
// over local deletions, and leaves the file itself alone: it is a read.
func TestSyncReload(t *testing.T) {
	a := syncApp(t)
	folder := t.TempDir()
	if err := a.SetSyncFolder(folder); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if _, err := a.Playlists.Create("Gone"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	document := `{"version":1,"playlists":{"items":[{"id":"local:kept","source":"local","name":"Kept","songs":[]}]}}`
	path := filepath.Join(folder, backup.BackupFile)
	if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
		t.Fatalf("write document: %v", err)
	}

	stats, err := a.SyncReload()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stats.Playlists != 1 {
		t.Fatalf("reload applied %d playlists, want 1", stats.Playlists)
	}
	list, _ := a.Playlists.List()
	names := make([]string, 0, len(list))
	for _, pl := range list {
		names = append(names, pl.Name)
	}
	if len(names) != 2 || !strings.Contains(strings.Join(names, ","), "Kept") {
		t.Fatalf("library = %v, want the local one plus the reloaded one", names)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != document {
		t.Fatalf("document = %q, %v; want it untouched", data, err)
	}
}

// A setting the document carries is local news too, so changing it must offer the
// manual save until a pass writes it out.
func TestSyncPendingSetting(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	// Autosave is on by default and kicks a pass on every setting write, which
	// would flush the change before the assertion. This test is about the switch
	// being off, where the pending mark is what offers the manual save.
	if err := a.SetSyncAuto(false); err != nil {
		t.Fatalf("disable auto sync: %v", err)
	}
	if err := a.SetSyncInterval(30); err != nil {
		t.Fatalf("set interval: %v", err)
	}
	if !a.SyncPending() {
		t.Fatal("a never synced folder reported nothing to sync")
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	if err := a.SetSyncInterval(45); err != nil {
		t.Fatalf("set interval: %v", err)
	}
	if !a.SyncPending() {
		t.Fatal("a changed setting was not reported as pending")
	}

	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if a.SyncPending() {
		t.Fatal("a setting the last pass wrote still reported a pending sync")
	}
}

// passes counts the background passes that reached the document.
type passes struct {
	count atomic.Int32
}

// broken is a section store that cannot be read, which fails the whole pass.
type broken struct{}

func (broken) Section() (backup.Settings, error) { return nil, os.ErrPermission }
func (broken) Apply(backup.Settings) error       { return nil }
func (broken) Pending() bool                     { return false }
func (broken) Synced()                           {}

func (p *passes) Section() (backup.Settings, error) {
	p.count.Add(1)
	return nil, nil
}

func (p *passes) Apply(backup.Settings) error { return nil }
func (p *passes) Pending() bool               { return false }
func (p *passes) Synced()                     {}

// idle waits for the debounce timer and the background loop to settle.
func idle(t *testing.T, a *App) {
	t.Helper()
	armed := func() bool {
		a.syncTimerMu.Lock()
		defer a.syncTimerMu.Unlock()
		return a.syncTimer != nil
	}
	deadline := time.Now().Add(5 * time.Second)
	for armed() || a.syncRunning.Load() || a.syncQueued.Load() {
		if time.Now().After(deadline) {
			t.Fatal("sync did not finish")
		}
		time.Sleep(time.Millisecond)
	}
}

// A burst of changes asks for as many syncs as it makes, but one pass is enough
// because they all read the same stores: the queue keeps a single pending pass.
func TestSyncCoalescesBurst(t *testing.T) {
	debounce, maxWait := syncDebounce, syncMaxWait
	syncDebounce, syncMaxWait = 10*time.Millisecond, time.Second
	t.Cleanup(func() { syncDebounce, syncMaxWait = debounce, maxWait })
	a := syncApp(t)
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if err := settings.SetObject("sync-auto", true); err != nil {
		t.Fatalf("enable auto sync: %v", err)
	}
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	counted := &passes{}
	a.Backup.Settings = counted

	// Hold the loop from taking the queued pass so the whole burst lands before
	// it starts, which is the case the single flag exists for.
	a.syncRunning.Store(true)
	for range 20 {
		a.syncInBackground()
	}
	a.syncRunning.Store(false)
	a.syncInBackground()

	idle(t, a)
	if got := counted.count.Load(); got != 1 {
		t.Fatalf("burst ran %d passes, want 1", got)
	}
}

// A restore answers what it applied per section, so the notice can say what the
// file brought instead of only that something changed.
func TestImportBackupCountsSections(t *testing.T) {
	a := syncApp(t)
	data := `{"version":1,` +
		`"playlists":{"items":[{"id":"local:mix","source":"local","name":"Mix","songs":[]}]},` +
		`"follows":{"items":[{"id":"bob","at":1}]},` +
		`"saved":{"items":[{"id":"spotify:playlist:x","name":"Liked","at":1}]},` +
		`"order":[{"scope":"library","account":"alice","ids":["a","b"],"at":1}],` +
		`"settings":{"sync":{"value":{"interval":30},"at":1}}}`

	stats, err := a.ImportBackup(data)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if stats.Playlists != 1 || stats.Follows != 1 || stats.Saved != 1 || stats.Order != 1 || stats.Settings != 1 {
		t.Fatalf("stats = %+v, want one record per section", stats)
	}
	if stats.Changed() != 5 {
		t.Fatalf("changed = %d, want 5", stats.Changed())
	}

	if list, _ := a.Playlists.List(); len(list) != 1 || list[0].Name != "Mix" {
		t.Fatalf("playlists = %+v", list)
	}
	if users, _ := a.Follows.List(); len(users) != 1 || users[0] != "bob" {
		t.Fatalf("follows = %v", users)
	}
	if kept, _ := a.Saved.List(); len(kept) != 1 || kept[0].Id != "spotify:playlist:x" {
		t.Fatalf("saved = %+v", kept)
	}
	if order, _ := a.Order.List(ordering.Library, "alice"); len(order) != 2 || order[0] != "a" {
		t.Fatalf("order = %v", order)
	}
	if got := a.SyncInterval(); got != 30 {
		t.Fatalf("interval = %d, want 30", got)
	}
}

// A copy a provider serves is written back only when it is not provably stale.
// The rule lives in the backend because the backend is what remembers our last
// write: the reading side only knows the bytes and the provider's stamp.
func TestSyncDocumentFreshness(t *testing.T) {
	const foreign = `{"version":1,"playlists":{"items":[{"id":"local:remote","source":"local","name":"Remote","songs":[]}]}}`

	// seed leaves a document this device wrote and a playlist the document does
	// not carry, so a fresh pass has local news to push out.
	seed := func(t *testing.T) (*App, string, int64) {
		t.Helper()
		a := syncApp(t)
		if _, err := a.Playlists.Create("Mix"); err != nil {
			t.Fatalf("create playlist: %v", err)
		}
		folder := t.TempDir()
		if err := a.SetSyncFolder(folder); err != nil {
			t.Fatalf("set folder: %v", err)
		}
		if err := a.SyncPlaylists(); err != nil {
			t.Fatalf("sync: %v", err)
		}
		own, err := os.ReadFile(filepath.Join(folder, backup.BackupFile))
		if err != nil {
			t.Fatalf("read document: %v", err)
		}
		at := a.SyncStatus().WriteAt
		if at == 0 {
			t.Fatal("a pass that wrote the document recorded no write")
		}
		if _, err := a.Playlists.Create("B Sides"); err != nil {
			t.Fatalf("create playlist: %v", err)
		}
		return a, string(own), at
	}

	cases := []struct {
		name         string
		document     func(own string) string
		lastModified func(at int64) int64
		trust        bool
		want         bool
	}{
		{
			name:         "our own bytes carry local news even when the stamp did not move",
			document:     func(own string) string { return own },
			lastModified: func(at int64) int64 { return at - 1 },
			want:         true,
		},
		{
			name:         "bytes that are not ours and did not move after our write are stale",
			document:     func(string) string { return foreign },
			lastModified: func(at int64) int64 { return at - 1 },
			want:         false,
		},
		{
			name:         "bytes newer than our write are fresh",
			document:     func(string) string { return foreign },
			lastModified: func(at int64) int64 { return at + 1 },
			want:         true,
		},
		{
			name:         "bytes without a timestamp are fresh",
			document:     func(string) string { return foreign },
			lastModified: func(int64) int64 { return 0 },
			want:         true,
		},
		{
			name:         "a manual save trusts the copy it read",
			document:     func(string) string { return foreign },
			lastModified: func(at int64) int64 { return at - 1 },
			trust:        true,
			want:         true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, own, at := seed(t)
			_, write, err := a.syncDocument(c.document(own), c.lastModified(at), c.trust)
			if err != nil {
				t.Fatalf("document: %v", err)
			}
			if write != c.want {
				t.Fatalf("write = %v, want %v", write, c.want)
			}
		})
	}
}

// A folder this device never reconciled carries no baseline to judge a copy
// against, or an install pointed at a second folder would sit on another
// device's document forever.
func TestSyncDocumentFreshInAnotherFolder(t *testing.T) {
	a := syncApp(t)
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	at := a.SyncStatus().WriteAt
	if at == 0 {
		t.Fatal("a pass that wrote the document recorded no write")
	}

	// The new folder makes the copy look older than our own write, which is the
	// case the freshness rule exists for; the missing baseline must win.
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set second folder: %v", err)
	}
	_, write, err := a.syncDocument(`{"version":1,"playlists":{"items":[{"id":"local:remote","source":"local","name":"Remote"}]}}`, at-1, false)
	if err != nil {
		t.Fatalf("document: %v", err)
	}
	if !write {
		t.Fatal("a copy in a folder with no baseline of its own was treated as stale")
	}
}

// A copy that will not parse is what a provider serves while another device is
// still uploading it, or after a write that was cut short. Both platforms rebuild
// the document from this device instead of failing every later pass.
func TestSyncDocumentRecoversUnreadableCopy(t *testing.T) {
	const truncated = `{"version":1,"playlists":{"items":[{"id":"local:remote"`

	a := syncApp(t)
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}

	if a.SyncStatus().Problem != "" {
		t.Fatal("a failed pass left no reason in the status")
	}

	document, write, err := a.syncDocument(truncated, 1, false)
	if err != nil {
		t.Fatalf("automatic pass: %v", err)
	}
	if !write {
		t.Fatal("an automatic pass over an unreadable copy wrote nothing")
	}
	if !strings.Contains(document, "Mix") {
		t.Fatalf("the recovered document lost this device's library: %.120s", document)
	}
	if a.SyncStatus().Problem != "" {
		t.Fatal("a recovered pass kept the failure in the status")
	}
	if got := a.SyncStatus().Bytes; got != len(truncated) {
		t.Fatalf("the panel reports %d bytes for the copy it read, want %d", got, len(truncated))
	}
}

// A document this device never wrote carries no baseline to judge it against, so
// a first sync on a fresh install must push its news out instead of freezing.
func TestSyncDocumentFreshWithoutWrite(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}

	_, write, err := a.syncDocument(`{"version":1,"playlists":{"items":[{"id":"local:remote","source":"local","name":"Remote"}]}}`, 1, false)
	if err != nil {
		t.Fatalf("document: %v", err)
	}
	if !write {
		t.Fatal("a document never written here was treated as stale")
	}
}

// The document moves as a file: the platform stages the copy in incoming.json
// and reads the answer from outgoing.json. A pass with local news must write
// that answer, and a pass that changed nothing must write none, or the provider
// is asked to upload the copy it already has.
func TestSyncDocumentStagesFiles(t *testing.T) {
	a := syncApp(t)
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	dir := t.TempDir()
	incoming := filepath.Join(dir, syncIncoming)
	outgoing := filepath.Join(dir, syncOutgoing)

	const foreign = `{"version":1,"playlists":{"items":[]}}`
	if err := os.WriteFile(incoming, []byte(foreign), 0o600); err != nil {
		t.Fatalf("stage copy: %v", err)
	}
	write, err := a.syncDocumentAt(dir, 1, true)
	if err != nil {
		t.Fatalf("document: %v", err)
	}
	if !write {
		t.Fatal("a copy with local news answered nothing to write")
	}
	answer, err := os.ReadFile(outgoing)
	if err != nil {
		t.Fatalf("read answer: %v", err)
	}
	if !strings.Contains(string(answer), "Mix") {
		t.Fatalf("the answer lost this device's library: %.120s", answer)
	}

	// The copy already is the answer: nothing to write and no leftover stage.
	if err := os.WriteFile(incoming, answer, 0o600); err != nil {
		t.Fatalf("stage answer: %v", err)
	}
	if err := os.Remove(outgoing); err != nil {
		t.Fatalf("clear answer: %v", err)
	}
	write, err = a.syncDocumentAt(dir, 2, true)
	if err != nil {
		t.Fatalf("document: %v", err)
	}
	if write {
		t.Fatal("a settled copy answered a write")
	}
	if _, err := os.Stat(outgoing); !os.IsNotExist(err) {
		t.Fatal("a settled copy left an answer to upload")
	}
	if _, err := os.Stat(incoming); !os.IsNotExist(err) {
		t.Fatal("a settled copy left the stage behind")
	}
}

// The stamp a platform pass reconciles is the provider's, and the folder path
// has none to offer: a pass over the folder must not leave behind the stamp of a
// copy that pass never read, nor move the baseline it never wrote.
func TestSyncDocumentRecordsReadStamp(t *testing.T) {
	a := syncApp(t)
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	folder := t.TempDir()
	if err := a.SetSyncFolder(folder); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	written := a.SyncStatus().WriteAt
	if written == 0 {
		t.Fatal("a pass that wrote the document recorded no write")
	}
	own, err := os.ReadFile(filepath.Join(folder, backup.BackupFile))
	if err != nil {
		t.Fatalf("read document: %v", err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, syncIncoming), own, 0o600); err != nil {
		t.Fatalf("stage copy: %v", err)
	}
	if _, err := a.syncDocumentAt(dir, 1234, true); err != nil {
		t.Fatalf("document: %v", err)
	}
	if got := a.SyncStatus().ReadStamp; got != 1234 {
		t.Fatalf("read stamp = %d, want the provider's 1234", got)
	}

	if err := a.SyncSave(); err != nil {
		t.Fatalf("folder pass: %v", err)
	}
	status := a.SyncStatus()
	if status.ReadStamp != 0 {
		t.Fatalf("a folder pass kept the provider stamp %d it never read", status.ReadStamp)
	}
	if status.WriteAt != written {
		t.Fatalf("a folder pass that changed nothing moved the baseline from %d to %d", written, status.WriteAt)
	}
}

// A copy that will not parse is what a provider serves while another device is
// still uploading it, or after a write that was cut short. The answer still
// carries the local playlists instead of failing the file forever.
func TestSyncDocumentRecoversUnreadableFile(t *testing.T) {
	a := syncApp(t)
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, syncIncoming), []byte(`{"version":1,"playlists":`), 0o600); err != nil {
		t.Fatalf("stage copy: %v", err)
	}

	write, err := a.syncDocumentAt(dir, 1, false)
	if err != nil {
		t.Fatalf("automatic pass over an unreadable copy: %v", err)
	}
	if !write {
		t.Fatal("an automatic pass over an unreadable copy wrote nothing")
	}
	answer, err := os.ReadFile(filepath.Join(dir, syncOutgoing))
	if err != nil {
		t.Fatalf("read answer: %v", err)
	}
	if !strings.Contains(string(answer), "Mix") {
		t.Fatalf("the recovered answer lost this device's library: %.120s", answer)
	}
}

// The baseline is what the freshness rule compares a served copy against, so it
// may only move once the platform confirmed the commit. Otherwise a commit that
// failed would make every later pass refuse to write these bytes again.
func TestSyncWrittenMovesTheBaseline(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, syncIncoming), []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatalf("stage copy: %v", err)
	}
	write, err := a.syncDocumentAt(dir, 1, true)
	if err != nil {
		t.Fatalf("document: %v", err)
	}
	if !write {
		t.Fatal("a copy with local news answered nothing to write")
	}
	if at := a.SyncStatus().WriteAt; at != 0 {
		t.Fatalf("a pass that had not committed yet moved the baseline to %d", at)
	}
	answer, err := os.ReadFile(filepath.Join(dir, syncOutgoing))
	if err != nil {
		t.Fatalf("read answer: %v", err)
	}

	if err := a.syncWrittenAt(dir); err != nil {
		t.Fatalf("commit: %v", err)
	}
	status := a.SyncStatus()
	if status.WriteAt == 0 {
		t.Fatal("the committed document left no baseline")
	}
	if status.Hash != stamp(string(answer)) {
		t.Fatalf("baseline fingerprints %d, want the committed bytes %d", status.Hash, stamp(string(answer)))
	}
	if status.Stale || status.Problem != "" {
		t.Fatalf("a committed document kept stale=%v problem=%q", status.Stale, status.Problem)
	}
	if _, err := os.Stat(filepath.Join(dir, syncOutgoing)); !os.IsNotExist(err) {
		t.Fatal("the committed answer was left behind")
	}
}

// A pass that failed has nothing to retry by itself on the platform path, so the
// panel must offer the manual save for it.
func TestSyncPendingAfterProblem(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
	}
	if _, err := a.Playlists.Create("Mix"); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if a.SyncPending() {
		t.Fatal("a settled pass still offered a manual save")
	}

	a.recordProblem(errors.New("write failed"))
	if !a.SyncPending() {
		t.Fatal("a failed pass offered no manual save")
	}
}
