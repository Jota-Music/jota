package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/follows"
	"github.com/Jota-Music/jota/internal/ordering"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/rooms"
	"github.com/Jota-Music/jota/internal/services/discord/saved"
	"github.com/Jota-Music/jota/internal/state"
)

// TestMigrateLegacyStore imports the old store and checks that every section
// arrives in the shape the app reads.
func TestMigrateLegacyStore(t *testing.T) {
	dir := setupLegacy(t)

	if summary, err := MigrateLegacyStore(); err != nil {
		t.Fatal(err)
	} else if summary == "no legacy store found" {
		t.Fatal("the store was not found")
	}

	stored, _ := state.Read[playlists.Stored](state.Playlists)
	if len(stored.Tracks) != 2 {
		t.Fatalf("playlists: %v", stored.Tracks)
	}
	if len(stored.Index) != 2 {
		t.Fatalf("index: %v", stored.Index)
	}
	if _, ok := stored.Deleted["local:gone"]; !ok {
		t.Fatalf("graves: %v", stored.Deleted)
	}

	follow, _ := state.Read[follows.Section](state.Follows)
	if len(follow.Items) != 1 || follow.Items[0].Id != "youtube:channel1" {
		t.Fatalf("follows: %v", follow.Items)
	}

	kept, _ := state.Read[saved.Section](state.Saved)
	if _, ok := kept.Removed["youtube:playlist:gone"]; !ok {
		t.Fatalf("saved: %v", kept.Removed)
	}

	order, _ := state.Read[ordering.Section](state.Order)
	shelves := map[string]int{}
	for _, entry := range order {
		shelves[entry.Scope+"|"+entry.Account] = len(entry.Ids)
	}
	if shelves["library|"] != 1 {
		t.Fatalf("order: %v", order)
	}
	if shelves["library|user"] != 2 {
		t.Fatalf("order: %v", order)
	}
	// The store kept this shelf twice (a bare list and the record that replaced
	// it): the record wins and the shelf is one.
	if shelves["library|tifieme"] != 2 {
		t.Fatalf("order: %v", order)
	}
	if len(shelves) != 3 {
		t.Fatalf("shelves: %v", shelves)
	}

	rooms, _ := state.Read[[]rooms.Room](state.Rooms)
	if len(rooms) != 1 || rooms[0].Code != "family" {
		t.Fatalf("rooms: %v", rooms)
	}

	sessions, _ := state.Read[map[string]sessionEntry](state.Sessions)
	if sessions["youtube"].Value != "JOTAENC1:youtube-blob" {
		t.Fatalf("sessions: %v", sessions)
	}
	if sessions["spotify"].Value != "JOTAENC1:spotify-blob" {
		t.Fatalf("sessions: %v", sessions)
	}

	settings, _ := state.Read[deviceSettings](state.Settings)
	if settings.SessionPass != "hunter2" {
		t.Fatalf("settings: %v", settings)
	}
	if settings.Relay.URL != "relay.example" {
		t.Fatalf("settings: %v", settings)
	}
	if settings.Sync.Folder != "/home/user/sync" {
		t.Fatalf("settings: %v", settings)
	}

	spotify, _ := state.Read[spotifyAccount](state.Spotify)
	if spotify.Session == nil || spotify.Session.Username != "user" {
		t.Fatalf("spotify: %v", spotify)
	}
	if spotify.Device != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("spotify: %v", spotify)
	}

	youtube, _ := state.Read[youtubeAccount](state.YouTube)
	if youtube.Links["spotify:track:trackid1"] != "videoid11" {
		t.Fatalf("youtube: %v", youtube.Links)
	}

	// The window geometry goes to the file next to the log.
	window, err := os.ReadFile(filepath.Join(dir, "jota", "window.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(window) == 0 {
		t.Fatal("the window geometry is empty")
	}

	// The old store is out of the way: it was imported, checked and kept aside
	// so a mapping bug found later can be imported again.
	if HasLegacyStore() {
		t.Fatal("the store is still in place")
	}
	if _, err := os.Stat(filepath.Join(dir, "jota", "storage", "kv.badger")); err != nil {
		t.Fatalf("the store was not kept aside: %v", err)
	}
}

// TestPruneStore checks the kept store leaves on its own clock: an import from
// today stays, one whose time is up goes.
func TestPruneStore(t *testing.T) {
	t.Run("kept", func(t *testing.T) {
		dir := setupLegacy(t)
		if _, err := MigrateLegacyStore(); err != nil {
			t.Fatal(err)
		}
		pruneStore()
		if _, err := os.Stat(filepath.Join(dir, "jota", "storage", "kv.badger")); err != nil {
			t.Fatalf("a store imported today was removed: %v", err)
		}
	})

	t.Run("removed", func(t *testing.T) {
		dir := setupLegacy(t)
		if _, err := MigrateLegacyStore(); err != nil {
			t.Fatal(err)
		}
		aside := filepath.Join(dir, "jota", "storage", "kv.badger")
		old := time.Now().Add(-keepStore - time.Hour)
		if err := os.Chtimes(aside, old, old); err != nil {
			t.Fatal(err)
		}
		pruneStore()
		if _, err := os.Stat(aside); !os.IsNotExist(err) {
			t.Fatalf("the kept store outlived its time: %v", err)
		}
	})
}

// TestMigrateKeepsWindow checks that a retried import does not move a window the
// user already placed: window.json is the app's file, not the store's.
func TestMigrateKeepsWindow(t *testing.T) {
	dir := setupLegacy(t)
	path := filepath.Join(dir, "jota", "window.json")
	placed := []byte(`{"X":1,"Y":2,"Width":300,"Height":200}`)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, placed, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := MigrateLegacyStore(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(placed) {
		t.Fatalf("the window was replaced: %s", got)
	}
}

// TestMigrateKeepsWhatIsNewer checks that the state the device holds after the
// upgrade is not undone by the old store.
func TestMigrateKeepsWhatIsNewer(t *testing.T) {
	setupLegacy(t)

	// The device already holds a playlist of its own.
	kept := playlists.Stored{
		Index:  []string{"local:mine"},
		Tracks: map[string]playlists.Playlist{"local:mine": {Id: "local:mine", Name: "Mine"}},
	}
	if err := state.Write(state.Playlists, kept); err != nil {
		t.Fatal(err)
	}
	if err := state.Write(state.Follows, follows.Section{Items: []follows.Entry{{Id: "youtube:mine", At: 1}}}); err != nil {
		t.Fatal(err)
	}

	if _, err := MigrateLegacyStore(); err != nil {
		t.Fatal(err)
	}

	stored, _ := state.Read[playlists.Stored](state.Playlists)
	if _, ok := stored.Tracks["local:mine"]; !ok {
		t.Fatalf("the playlist of the device is gone: %v", stored.Tracks)
	}
	if _, ok := stored.Tracks["local:one"]; !ok {
		t.Fatalf("the playlist of the store is missing: %v", stored.Tracks)
	}

	follow, _ := state.Read[follows.Section](state.Follows)
	ids := map[string]bool{}
	for _, item := range follow.Items {
		ids[item.Id] = true
	}
	if !ids["youtube:mine"] || !ids["youtube:channel1"] {
		t.Fatalf("follows: %v", follow.Items)
	}
}

// TestMigrateMergesSavedTombstones checks that a removal the store knew is not
// lost because the device already had one. It used to skip the merge, fail the
// verification, and keep the old store for the next launch.
func TestMigrateMergesSavedTombstones(t *testing.T) {
	setupLegacy(t)

	kept := saved.Section{Removed: map[string]int64{"youtube:playlist:mine": 1700000000000}}
	if err := state.Write(state.Saved, kept); err != nil {
		t.Fatal(err)
	}

	if _, err := MigrateLegacyStore(); err != nil {
		t.Fatal(err)
	}

	section, _ := state.Read[saved.Section](state.Saved)
	if _, ok := section.Removed["youtube:playlist:mine"]; !ok {
		t.Fatalf("the tombstone of the device is gone: %v", section.Removed)
	}
	if _, ok := section.Removed["youtube:playlist:gone"]; !ok {
		t.Fatalf("the tombstone of the store is missing: %v", section.Removed)
	}
	if HasLegacyStore() {
		t.Fatal("the store is still there")
	}
}

// TestMigrateKeepsOnePlaylistPerName checks the case a local id is regenerated:
// the store holds another copy of a playlist the library already has, under a
// different id, and importing it would show the same name twice.
func TestMigrateKeepsOnePlaylistPerName(t *testing.T) {
	// The library's copy is newer, so the store's copy is not taken.
	t.Run("library newer", func(t *testing.T) {
		setupLegacy(t)
		kept := playlists.Stored{
			Index: []string{"local:mine"},
			Tracks: map[string]playlists.Playlist{
				"local:mine": {Id: "local:mine", Name: "One", Songs: []string{"youtube:mine"}, ModifiedAt: 1794268407000},
			},
		}
		if err := state.Write(state.Playlists, kept); err != nil {
			t.Fatal(err)
		}

		if _, err := MigrateLegacyStore(); err != nil {
			t.Fatal(err)
		}

		stored, _ := state.Read[playlists.Stored](state.Playlists)
		if _, ok := stored.Tracks["local:mine"]; !ok {
			t.Fatalf("the library's copy is gone: %v", stored.Tracks)
		}
		if _, ok := stored.Tracks["local:one"]; ok {
			t.Fatalf("the same playlist arrived twice: %v", stored.Tracks)
		}
		if got := stored.Tracks["local:mine"].Songs; len(got) != 1 || got[0] != "youtube:mine" {
			t.Fatalf("the newer copy was replaced: %v", got)
		}
		if slices.Contains(stored.Index, "local:one") {
			t.Fatalf("the dropped copy is in the shelf: %v", stored.Index)
		}
		if len(stored.Tracks) != 2 {
			t.Fatalf("playlists: %v", stored.Tracks)
		}
	})

	// The store's copy is newer, so its content wins, but it keeps the id the
	// library already uses: the shelf order and the tombstones still point at it.
	t.Run("store newer", func(t *testing.T) {
		setupLegacy(t)
		kept := playlists.Stored{
			Index: []string{"local:mine"},
			Tracks: map[string]playlists.Playlist{
				"local:mine": {Id: "local:mine", Name: "One", Songs: []string{"youtube:old"}, ModifiedAt: 1600000000000},
			},
		}
		if err := state.Write(state.Playlists, kept); err != nil {
			t.Fatal(err)
		}

		if _, err := MigrateLegacyStore(); err != nil {
			t.Fatal(err)
		}

		stored, _ := state.Read[playlists.Stored](state.Playlists)
		if _, ok := stored.Tracks["local:one"]; ok {
			t.Fatalf("the same playlist arrived twice: %v", stored.Tracks)
		}
		if got := stored.Tracks["local:mine"].Songs; len(got) != 1 || got[0] != "youtube:aaaaaaaaaaa" {
			t.Fatalf("the newer copy did not win: %v", got)
		}
		if len(stored.Tracks) != 2 {
			t.Fatalf("playlists: %v", stored.Tracks)
		}
	})
}

// TestMigrateKeepsYoutubeLinks checks that the links the store holds join the
// ones the device already has. It used to take them only when the device had
// none, so a device that had played a few songs since the upgrade dropped the
// rest of them silently, and the verification could not see it.
func TestMigrateKeepsYoutubeLinks(t *testing.T) {
	setupLegacy(t)

	kept := youtubeAccount{Links: map[string]string{"spotify:track:mine": "youtube:mine"}}
	if err := state.Write(state.YouTube, kept); err != nil {
		t.Fatal(err)
	}

	if _, err := MigrateLegacyStore(); err != nil {
		t.Fatal(err)
	}

	got, _ := state.Read[youtubeAccount](state.YouTube)
	if got.Links["spotify:track:mine"] != "youtube:mine" {
		t.Fatalf("the link of the device is gone: %v", got.Links)
	}
	if got.Links["spotify:track:trackid1"] != "videoid11" {
		t.Fatalf("the link of the store is missing: %v", got.Links)
	}
	if len(got.Links) != 2 {
		t.Fatalf("links: %v", got.Links)
	}
}

// TestMigrateKeepsTombstones checks that what this device removed is not
// brought back by the store, which still holds the copy that was dropped. The
// store is deleted after the import, so a record it restores stays restored.
func TestMigrateKeepsTombstones(t *testing.T) {
	t.Run("playlist", func(t *testing.T) {
		setupLegacy(t)
		if err := state.Write(state.Playlists, playlists.Stored{
			Deleted: map[string]int64{"local:one": 1700000000100},
		}); err != nil {
			t.Fatal(err)
		}

		if _, err := MigrateLegacyStore(); err != nil {
			t.Fatal(err)
		}

		stored, _ := state.Read[playlists.Stored](state.Playlists)
		if _, ok := stored.Tracks["local:one"]; ok {
			t.Fatalf("the removed playlist came back: %v", stored.Tracks)
		}
		if slices.Contains(stored.Index, "local:one") {
			t.Fatalf("the removed playlist is in the shelf: %v", stored.Index)
		}
		if stored.Deleted["local:one"] != 1700000000100 {
			t.Fatalf("the tombstone changed: %v", stored.Deleted)
		}
		if _, ok := stored.Tracks["local:two"]; !ok {
			t.Fatalf("the playlist the store kept is missing: %v", stored.Tracks)
		}
	})

	t.Run("follow", func(t *testing.T) {
		setupLegacy(t)
		if err := state.Write(state.Follows, follows.Section{
			Removed: map[string]int64{"youtube:channel1": 1700000000100},
		}); err != nil {
			t.Fatal(err)
		}

		if _, err := MigrateLegacyStore(); err != nil {
			t.Fatal(err)
		}

		section, _ := state.Read[follows.Section](state.Follows)
		for _, item := range section.Items {
			if item.Id == "youtube:channel1" {
				t.Fatalf("the unfollowed channel came back: %v", section.Items)
			}
		}
		if section.Removed["youtube:channel1"] != 1700000000100 {
			t.Fatalf("the tombstone changed: %v", section.Removed)
		}
	})

	t.Run("saved", func(t *testing.T) {
		setupLegacy(t)
		if err := state.Write(state.Saved, saved.Section{
			Removed: map[string]int64{"youtube:playlist:kept": 1700000000100},
		}); err != nil {
			t.Fatal(err)
		}

		if _, err := MigrateLegacyStore(); err != nil {
			t.Fatal(err)
		}

		section, _ := state.Read[saved.Section](state.Saved)
		for _, item := range section.Items {
			if item.Id == "youtube:playlist:kept" {
				t.Fatalf("the unsaved playlist came back: %v", section.Items)
			}
		}
		if section.Removed["youtube:playlist:kept"] != 1700000000100 {
			t.Fatalf("the tombstone changed: %v", section.Removed)
		}
	})
}

// TestImportIsRepeatable checks that the same store answers the same summary
// every time, whatever order its maps are walked in. It caught a shelf the old
// store kept twice being counted twice half of the runs.
func TestImportIsRepeatable(t *testing.T) {
	var summary string
	for i := 0; i < 20; i++ {
		setupLegacy(t)
		got, err := MigrateLegacyStore()
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			summary = got
			continue
		}
		if got != summary {
			t.Fatalf("the summary varies: %q and %q", summary, got)
		}
	}
}

// TestImportOnStartup runs the flow the app runs at launch: with no one
// calling anything, the old store is imported and removed, and the page is
// told to read the library again.
func TestImportOnStartup(t *testing.T) {
	setupLegacy(t)

	importLegacyStore()

	stored, _ := state.Read[playlists.Stored](state.Playlists)
	if len(stored.Tracks) != 2 {
		t.Fatalf("playlists: %v", stored.Tracks)
	}
	follow, _ := state.Read[follows.Section](state.Follows)
	if len(follow.Items) != 1 {
		t.Fatalf("follows: %v", follow.Items)
	}
	if HasLegacyStore() {
		t.Fatal("the store is still there")
	}
}

// TestImportOnStartupWithoutStore checks that a device that never had the old
// store is left alone: one stat, nothing written, nothing logged.
func TestImportOnStartupWithoutStore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("APPDATA", dir)
	state.Reset()
	t.Cleanup(state.Reset)

	importLegacyStore()

	if _, err := os.Stat(filepath.Join(dir, "jota", "state.json")); !os.IsNotExist(err) {
		t.Fatalf("a state document was written: %v", err)
	}
}

// TestImportOnStartupKeepsDamagedStore checks that a store the reader refuses
// is not deleted: the import fails, the store stays for the next launch.
func TestImportOnStartupKeepsDamagedStore(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("APPDATA", dir)
	state.Reset()
	t.Cleanup(state.Reset)

	store := filepath.Join(dir, "jota", "storage", "kv")
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatal(err)
	}
	// The magic and the version are right, the change set does not match its
	// checksum: a damaged store, not an empty one.
	manifest := []byte("Bdgr\x00\x00\x00\x08" + "\x00\x00\x00\x04" + "\xde\xad\xbe\xef" + "\x00\x00\x00\x00")
	if err := os.WriteFile(filepath.Join(store, "MANIFEST"), manifest, 0o600); err != nil {
		t.Fatal(err)
	}

	importLegacyStore()

	if !HasLegacyStore() {
		t.Fatal("the damaged store was removed")
	}
}

// setupLegacy puts the old store in a throwaway config folder and points the
// app at it.
func setupLegacy(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("APPDATA", dir)
	state.Reset()
	t.Cleanup(state.Reset)

	store := filepath.Join(dir, "jota", "storage", "kv")
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"MANIFEST", "000001.sst"} {
		data, err := os.ReadFile(filepath.Join("testdata", "store", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(store, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}
