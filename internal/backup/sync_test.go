package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/playlists"
)

func setup(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)
}

// item is one playlist as a document writes it.
func item(id, name string, at int64, songs ...string) string {
	refs := make([]string, 0, len(songs))
	for _, song := range songs {
		refs = append(refs, `{"id":"`+song+`"}`)
	}
	return fmt.Sprintf(`{"id":%q,"source":"local","name":%q,"songs":[%s],"modifiedAt":%d}`, id, name, strings.Join(refs, ","), at)
}

// write puts a document whose playlists section holds the given items in the
// folder, the way another device would have left it.
func write(t *testing.T, path, section string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(`{"version":1,"playlists":`+section+`}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeItems(t *testing.T, path string, items ...string) {
	t.Helper()
	write(t, path, `{"items":[`+strings.Join(items, ",")+`]}`)
}

// mirror is the shape of the document on disk, read back loosely enough to
// assert what was written without depending on the package's own writer.
type mirror struct {
	Version   int `json:"version"`
	Playlists struct {
		Items []struct {
			Id         string `json:"id"`
			Source     string `json:"source"`
			Name       string `json:"name"`
			ModifiedAt int64  `json:"modifiedAt"`
			Songs      []struct {
				Id string `json:"id"`
			} `json:"songs"`
		} `json:"items"`
		Deleted map[string]int64 `json:"deleted"`
		Songs   map[string]struct {
			Name string `json:"name"`
		} `json:"songs"`
	} `json:"playlists"`
}

func readDoc(t *testing.T, path string) mirror {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc mirror
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func ids(doc mirror) []string {
	out := make([]string, 0, len(doc.Playlists.Items))
	for _, p := range doc.Playlists.Items {
		out = append(out, p.Id)
	}
	return out
}

// The newer copy wins the track list, but metadata never disappears just
// because the other device had not resolved it yet.
func TestSyncKeepsMetadataFromOlderCopy(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	path := filepath.Join(dir, BackupFile)
	s := playlists.New(func(id string) (music.Song, error) {
		return music.Song{Id: id, Name: "Known"}, nil
	})
	summary, err := s.Create("Local")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddSongs(summary.Id, []string{"spotify:track:a"}); err != nil {
		t.Fatal(err)
	}

	writeItems(t, path, item(summary.Id, "Local", time.Now().Add(time.Hour).UnixMilli(), "spotify:track:a"))
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}

	doc := readDoc(t, path)
	if doc.Playlists.Songs["spotify:track:a"].Name != "Known" {
		t.Fatalf("file lost metadata: %+v", doc.Playlists.Songs)
	}
	section, err := s.ExportSection(nil)
	if err != nil {
		t.Fatal(err)
	}
	if stored := section.Items[0].Tracks["spotify:track:a"]; stored.Name != "Known" {
		t.Fatalf("store lost metadata: %+v", section.Items)
	}
}

// A renamed backup is still picked up: the canonical name is only a hint, the
// document shape is what makes a file the backup. It is then adopted under the
// canonical name, so the folder holds a single backup again.
func TestSyncUsesRenamedBackup(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "from-phone.json")
	writeItems(t, path, item("local:aaaaaaaa", "From phone", 10, "spotify:track:a"))

	s := playlists.New(nil)
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}

	list, err := s.List()
	if err != nil || len(list) != 1 || list[0].Name != "From phone" {
		t.Fatalf("pulled playlists = %+v, %v", list, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("renamed backup was not adopted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, BackupFile)); err != nil {
		t.Fatalf("sync left no canonical backup: %v", err)
	}
}

// A renamed backup is found by its content: the newest file that parses as a
// backup wins, and a file of another format is skipped instead of blocking the
// folder.
func TestSyncPicksNewestReadableBackup(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte(`{"version":9}`), 0o644); err != nil {
		t.Fatal(err)
	}
	older := filepath.Join(dir, "older.json")
	newer := filepath.Join(dir, "newer.json")
	writeItems(t, older, item("local:aaaaaaaa", "Older", 10))
	writeItems(t, newer, item("local:bbbbbbbb", "Newer", 10))
	// Filesystems keep coarser times than the writes here, so the order is set.
	for i, path := range []string{older, newer} {
		at := time.Unix(int64(1000+i), 0)
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}

	s := playlists.New(nil)
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}

	list, err := s.List()
	if err != nil || len(list) != 1 || list[0].Name != "Newer" {
		t.Fatalf("pulled = %+v, %v; want the newest readable backup", list, err)
	}
}

// An empty folder is a first setup: the local playlists become the backup.
func TestSyncEmptyFolderCreatesBackup(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	s := playlists.New(nil)
	summary, err := s.Create("Only local")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}
	doc := readDoc(t, filepath.Join(dir, BackupFile))
	if got := ids(doc); len(got) != 1 || got[0] != summary.Id {
		t.Fatalf("created backup = %v", got)
	}
}

// An empty folder with nothing to back up stays untouched, so a wrong folder is
// reported instead of being filled with an empty library.
func TestSyncWithoutBackupFails(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	s := playlists.New(nil)

	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err == nil {
		t.Fatal("sync without backup = nil, want error")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("sync left files in %s: %v", dir, entries)
	}
}

// An untimed remote copy has unknown age, so it only fills an equally untimed
// local copy that has nothing in it.
func TestSyncKeepsUntimedLocalWithSongs(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	path := filepath.Join(dir, BackupFile)
	s := playlists.New(nil)

	// Both copies are seeded from a document, so both are untimed.
	writeItems(t, path,
		item("local:empty", "Empty", 0),
		item("local:full", "Full", 0, "spotify:track:x"),
	)
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}

	writeItems(t, path,
		item("local:empty", "Empty", 0, "spotify:track:a"),
		item("local:full", "Full", 0, "spotify:track:b"),
	)
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}

	doc := readDoc(t, path)
	songs := map[string][]string{}
	for _, p := range doc.Playlists.Items {
		for _, ref := range p.Songs {
			songs[p.Id] = append(songs[p.Id], ref.Id)
		}
	}
	if want := []string{"spotify:track:a"}; !slices.Equal(songs["local:empty"], want) {
		t.Fatalf("empty playlist = %v, want %v", songs["local:empty"], want)
	}
	if want := []string{"spotify:track:x"}; !slices.Equal(songs["local:full"], want) {
		t.Fatalf("timer-less local lost its songs: %v, want %v", songs["local:full"], want)
	}
}

// A playlist deleted here must stay gone even though the backup still has it.
func TestSyncDoesNotResurrectDeleted(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	path := filepath.Join(dir, BackupFile)
	s := playlists.New(nil)
	summary, err := s.Create("Gone")
	if err != nil {
		t.Fatal(err)
	}
	writeItems(t, path, item(summary.Id, "Gone", 10))

	if err := s.Delete(summary.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}

	if list, _ := s.List(); len(list) != 0 {
		t.Fatalf("list after sync = %v, want empty", list)
	}
	if slices.Contains(ids(readDoc(t, path)), summary.Id) {
		t.Fatal("deleted playlist is back in the file")
	}
}

// Syncing an unchanged state must not touch the file, so file watchers and other
// devices do not see a write every interval.
func TestSyncSecondPassWritesNothing(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	path := filepath.Join(dir, BackupFile)
	writeItems(t, path, item("local:bbbbbbbb", "Pulled", 10))

	s := playlists.New(nil)
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	wrote := readDoc(t, path)

	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Fatalf("second sync rewrote the file: %v -> %v", before.ModTime(), after.ModTime())
	}
	if got := readDoc(t, path); !reflect.DeepEqual(got, wrote) {
		t.Fatalf("second sync changed the file: %+v -> %+v", wrote, got)
	}
}

// The document is published through a temporary file in the same folder, so no
// scratch file is left for the user or the sync tool to pick up.
func TestSyncWritesAtomically(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	path := filepath.Join(dir, BackupFile)
	writeItems(t, path, item("local:aaaaaaaa", "Pulled", 10))

	s := playlists.New(nil)
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("Local"); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != BackupFile {
		t.Fatalf("folder = %v, want only the backup", entries)
	}
	if got := ids(readDoc(t, path)); len(got) != 2 {
		t.Fatalf("document = %v, want both playlists", got)
	}
}

// Pulling creates playlists the store did not have, including ones whose id came
// from another instance.
func TestSyncPullsIntoStore(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	writeItems(t, filepath.Join(dir, BackupFile),
		item("local:cccccccc", "One", 5, "spotify:track:a"),
		item("other-device-id", "Two", 6),
	)

	s := playlists.New(nil)
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}

	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %+v, want 2 playlists", list)
	}
	names := []string{list[0].Name, list[1].Name}
	slices.Sort(names)
	if want := []string{"One", "Two"}; !slices.Equal(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
}

// A document another format left behind blocks a normal sync, but a forced one
// writes the store over it instead of leaving the folder stuck.
func TestSyncForceOverwritesUnreadableBackup(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	path := filepath.Join(dir, BackupFile)
	if err := os.WriteFile(path, []byte(`{"version":2,"playlists":{"items":[]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	s := playlists.New(nil)
	if _, err := s.Create("Mix"); err != nil {
		t.Fatal(err)
	}

	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err == nil {
		t.Fatal("sync read a document of another format")
	}

	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Force: true, Import: true}); err != nil {
		t.Fatalf("forced sync: %v", err)
	}
	doc := readDoc(t, path)
	if doc.Version != Version || len(doc.Playlists.Items) != 1 || doc.Playlists.Items[0].Name != "Mix" {
		t.Fatalf("forced document = %+v", doc)
	}
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatalf("sync after the forced write: %v", err)
	}
}

// A damaged document is not a reason to stop backing up: the pass writes this
// device's library over it, so autosave revives on its own.
func TestSyncRecoversUnreadableBackup(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	path := filepath.Join(dir, BackupFile)
	broken := `{"version":1,"playlists":{"items":[`
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	s := playlists.New(nil)
	if _, err := s.Create("Mix"); err != nil {
		t.Fatal(err)
	}

	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatalf("sync of a damaged document: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != BackupFile {
		t.Fatalf("folder = %v, want only the document", entries)
	}
	doc := readDoc(t, path)
	if doc.Version != Version || len(doc.Playlists.Items) != 1 || doc.Playlists.Items[0].Name != "Mix" {
		t.Fatalf("recovered document = %+v", doc)
	}
	list, err := s.List()
	if err != nil || len(list) != 1 || list[0].Name != "Mix" {
		t.Fatalf("library after the recovery = %+v, %v", list, err)
	}
}

// A document of a format this build does not write is not damage: writing over
// it would lose whatever that build stored, so the pass fails and leaves it.
func TestSyncKeepsForeignVersionBackup(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	path := filepath.Join(dir, BackupFile)
	foreign := `{"version":2,"playlists":{"items":[]}}`
	if err := os.WriteFile(path, []byte(foreign), 0o644); err != nil {
		t.Fatal(err)
	}

	s := playlists.New(nil)
	if _, err := s.Create("Mix"); err != nil {
		t.Fatal(err)
	}

	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true}); !errors.Is(err, ErrVersion) {
		t.Fatalf("sync = %v, want ErrVersion", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != foreign {
		t.Fatalf("foreign document = %q, %v; want it untouched", data, err)
	}
}

// The document is also the backup of the source-owned playlists: they are
// written once and then kept, so a pass costs no network while they are there.
func TestSyncCarriesSnapshots(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	s := playlists.New(nil)
	if _, err := s.Create("Mix"); err != nil {
		t.Fatal(err)
	}

	calls := 0
	snapshots := func() []playlists.Snapshot {
		calls++
		return []playlists.Snapshot{{
			Id:     "spotify:p1",
			Source: "spotify",
			Name:   "Liked",
			Songs:  []music.Song{{Id: "spotify:s1", Name: "Song"}},
		}}
	}

	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true, Snapshots: snapshots}); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if calls != 1 {
		t.Fatalf("resolved the sources %d times, want 1", calls)
	}
	doc := readDoc(t, filepath.Join(dir, BackupFile))
	if len(doc.Playlists.Items) != 2 {
		t.Fatalf("items = %+v", doc.Playlists.Items)
	}
	remote := doc.Playlists.Items[1]
	if remote.Source != "spotify" || remote.Name != "Liked" {
		t.Fatalf("external = %+v", remote)
	}

	// The snapshot is not local state: the store has one playlist, not two.
	list, err := s.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("list = %+v, %v", list, err)
	}

	// A second pass finds them in the file and does not download them again.
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true, Snapshots: snapshots}); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if calls != 1 {
		t.Fatalf("resolved the sources %d times, want still 1", calls)
	}

	// A refreshing pass (the snapshot aged, or the user saved by hand)
	// downloads them again even though the file already has them.
	stats, err := (&Service{Playlists: s}).Sync(dir, Options{Import: true, Refresh: true, Snapshots: snapshots})
	if err != nil {
		t.Fatalf("refreshing sync: %v", err)
	}
	if calls != 2 {
		t.Fatalf("resolved the sources %d times, want 2", calls)
	}
	if !stats.Snapshots {
		t.Fatal("a refreshing pass did not report the refreshed snapshots")
	}

	// A forced pass refreshes them too.
	if _, err := (&Service{Playlists: s}).Sync(dir, Options{Force: true, Import: true, Snapshots: snapshots}); err != nil {
		t.Fatalf("forced sync: %v", err)
	}
	if calls != 3 {
		t.Fatalf("resolved the sources %d times, want 3", calls)
	}
}

// With import off a pass still writes the union, so a playlist another device
// holds is never dropped from the file, but this library keeps its own records
// and a local deletion is not buried by a remote tombstone.
func TestSyncWithoutImportKeepsLibrary(t *testing.T) {
	setup(t)

	dir := t.TempDir()
	s := playlists.New(nil)
	own, err := s.Create("Own")
	if err != nil {
		t.Fatal(err)
	}
	writeItems(t, filepath.Join(dir, BackupFile), item("other-device-id", "Other", 20, "youtube:a"))

	stats, err := (&Service{Playlists: s}).Sync(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if stats.Playlists != 0 {
		t.Fatalf("imported = %d, want 0", stats.Playlists)
	}
	list, err := s.List()
	if err != nil || len(list) != 1 || list[0].Id != own.Id {
		t.Fatalf("library = %+v, %v", list, err)
	}
	if got := ids(readDoc(t, filepath.Join(dir, BackupFile))); len(got) != 2 {
		t.Fatalf("document = %v, want the union", got)
	}
}

// The Android document path and the desktop folder path must end at the same
// document, or a phone and a desktop would keep rewriting each other's backup.
func TestMergeMatchesSync(t *testing.T) {
	doc := `{"version":1,"playlists":{"items":[` +
		item("local:eeee", "Phone", 20, "youtube:a") + `,` +
		item("phone-id", "Other", 10, "youtube:b") +
		`],"deleted":{"local:gone":30}}}`

	setup(t)
	merged, stats, err := (&Service{Playlists: playlists.New(nil)}).ImportDocument(doc, nil, false, true)
	if err != nil {
		t.Fatalf("merge document: %v", err)
	}
	if stats.Playlists != 2 {
		t.Fatalf("imported = %d, want 2", stats.Playlists)
	}

	setup(t)
	dir := t.TempDir()
	path := filepath.Join(dir, BackupFile)
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (&Service{Playlists: playlists.New(nil)}).Sync(dir, Options{Import: true}); err != nil {
		t.Fatalf("sync: %v", err)
	}
	synced, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(synced) != merged {
		t.Fatalf("sync wrote\n%s\nmerge document answered\n%s", synced, merged)
	}
}
