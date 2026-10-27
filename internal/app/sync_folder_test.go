package app

import (
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
	if err := a.SyncPlaylists(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if got := counted.count.Load(); got != 0 {
		t.Fatalf("unchanged folder ran %d passes, want 0", got)
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

// A setting the document carries is local news too, so changing it must offer the
// manual save until a pass writes it out.
func TestSyncPendingSetting(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncFolder(t.TempDir()); err != nil {
		t.Fatalf("set folder: %v", err)
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

func (p *passes) Section() (backup.Settings, error) {
	p.count.Add(1)
	return nil, nil
}

func (p *passes) Apply(backup.Settings) error { return nil }
func (p *passes) Pending() bool               { return false }
func (p *passes) Synced()                     {}

// idle waits for the background loop to finish the pass it holds.
func idle(t *testing.T, a *App) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for a.syncRunning.Load() || a.syncQueued.Load() {
		if time.Now().After(deadline) {
			t.Fatal("sync did not finish")
		}
		time.Sleep(time.Millisecond)
	}
}

// A burst of changes asks for as many syncs as it makes, but one pass is enough
// because they all read the same stores: the queue keeps a single pending pass.
func TestSyncCoalescesBurst(t *testing.T) {
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
