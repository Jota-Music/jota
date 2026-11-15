package app

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/follows"
	"github.com/Jota-Music/jota/internal/legacy"
	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/ordering"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/rooms"
	"github.com/Jota-Music/jota/internal/services/discord/saved"
	"github.com/Jota-Music/jota/internal/state"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// legacyDir is where the old Badger store lived.
func legacyDir() (string, error) {
	base, err := state.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "jota", "storage", "kv"), nil
}

// HasLegacyStore reports whether an old Badger store is present and worth
// migrating. The import asks before reading anything, and `cmd/migrate` asks to
// say there is nothing to do.
func HasLegacyStore() bool {
	dir, err := legacyDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(dir, "MANIFEST"))
	return err == nil
}

// MigrateLegacyStore reads the old Badger store and writes its contents into
// the new state.json, then removes the old directory so it is not imported
// twice and the space it took is given back. It returns a short summary of
// what was imported.
//
// The cache is NOT migrated: tracks, artists, albums and the rest are
// re-fetched the next time the source answers. Only the device's own state
// (playlists, follows, ordering, rooms, saved, settings, sessions, spotify,
// youtube, window geometry) is carried over.
//
// The state the device already holds is newer than the store, so it wins where
// both have something and the store only fills the gaps: an upgrade never
// undoes what the user did afterwards.
//
// The old directory is only removed once the import has been read back and
// found complete: a half imported state keeps the copy it came from.
func MigrateLegacyStore() (string, error) {
	dir, err := legacyDir()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(dir, "MANIFEST")); err != nil {
		return "no legacy store found", nil
	}

	buckets, skipped, err := legacy.Read(dir)
	if err != nil {
		return "", fmt.Errorf("cannot read legacy store: %w", err)
	}
	if skipped > 0 {
		log.Printf("migrate: %d cached values live in the value log and are not read", skipped)
	}

	m := migration{buckets: buckets, expect: map[string]int{}}
	m.playlists()
	m.follows()
	m.saved()
	m.order()
	m.rooms()
	m.settings()
	m.sessions()
	m.spotify()
	m.youtube()
	m.window()

	// Put the document on disk before the old store is touched: the CLI has no
	// writer thread to flush it.
	state.Close()

	if err := m.verify(); err != nil {
		return "", fmt.Errorf("the import did not check out, the old store is kept: %w", err)
	}

	// Move the old store aside, so it is neither imported a second time nor
	// read again by anything that looks for a store to import.
	aside := dir + ".badger"
	if err := os.Rename(dir, aside); err != nil {
		// The import is repeatable, so a store that could not be moved is left
		// where it is and the next launch tries again.
		log.Printf("migrate: cannot move legacy store aside: %v", err)
		return fmt.Sprintf("imported %d records from legacy store", m.count), nil
	}
	// The clock starts now and not when the store was last written: a device
	// that had not opened the app in months is not an old import.
	now := time.Now()
	_ = os.Chtimes(aside, now, now)

	return fmt.Sprintf("imported %d records from legacy store", m.count), nil
}

// keepStore is how long the old store stays on disk after a successful import.
//
// verify counts what came across; it cannot prove the values are right. The
// store is the only copy of what the previous version knew, so it is not
// deleted by the import: a mapping bug found later is fixed and imported again.
// The cost of being wrong is the other way round: a fortnight of 17 MB against
// a library.
const keepStore = 14 * 24 * time.Hour

// pruneStore removes the store an earlier import kept aside once its time is
// up. With nothing kept aside it is one stat.
func pruneStore() {
	dir, err := legacyDir()
	if err != nil {
		return
	}
	aside := dir + ".badger"
	info, err := os.Stat(aside)
	if err != nil || time.Since(info.ModTime()) < keepStore {
		return
	}
	if freed := removeTree(aside); freed > 0 {
		log.Printf("migrate: the kept store is gone, %d MB given back", freed/(1<<20))
	}
	// The folder the store lived in is empty now, and nothing else uses it.
	_ = os.Remove(filepath.Dir(aside))
}

// removeTree deletes a directory and answers how many bytes it held.
func removeTree(dir string) int64 {
	var total int64
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("migrate: cannot remove the old store: %v", err)
		return 0
	}
	return total
}

// importLegacy is the import of the old store, done once per launch. Every task
// that reads a section it writes calls it first: staging the YouTube cookies,
// staging the Spotify session and connecting the account all read the very
// sections the import fills, and the store takes a moment to read. Without this
// they run first and come up empty, so an upgrade would import a session nobody
// uses until the next launch.
var importLegacy = sync.OnceFunc(importLegacyStore)

// importLegacyStore imports a store left on the device and tells the page to
// read the library again. It is safe on every launch: with no store it is two
// stats.
//
// It runs off the startup path, because reading a 17 MB store must not delay
// the window. A store that fails to import stays where it is, so the next
// launch tries again instead of silently dropping what it held.
func importLegacyStore() {
	// A store an earlier import kept aside leaves on its own schedule, whether
	// this launch imports anything or not.
	pruneStore()
	if !HasLegacyStore() {
		return
	}
	summary, err := MigrateLegacyStore()
	if err != nil {
		log.Printf("migrate: %v", err)
		return
	}
	log.Printf("migrate: %s", summary)
	if app := application.Get(); app != nil {
		app.Event.Emit("store:ready")
	}
}

// migration carries the old store into the new state document.
type migration struct {
	buckets map[string]map[string][]byte
	count   int
	// expect is how many records the old store held for each section, so the
	// import can be read back and checked.
	expect map[string]int
}

// verify reads the sections back and checks that what the old store held is
// there. The old store is only removed when this passes.
func (m *migration) verify() error {
	var missing []string
	for _, check := range []struct {
		name string
		got  int
	}{
		{"playlists", len(m.storedPlaylists())},
		{"follows", len(m.storedFollows().Items)},
		{"saved", len(m.storedSaved().Items) + len(m.storedSaved().Removed)},
		{"order", len(m.storedOrder())},
		{"rooms", len(m.storedRooms())},
		{"sessions", len(m.storedSessions())},
		{"spotify", m.storedSpotify()},
		{"youtube", len(m.storedYoutube().Links)},
	} {
		want := m.expect[check.name]
		if want == 0 {
			continue
		}
		if check.got < want {
			missing = append(missing, fmt.Sprintf("%s (%d of %d)", check.name, check.got, want))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("these sections came short: %s", strings.Join(missing, ", "))
	}
	return nil
}

func (m *migration) storedPlaylists() map[string]playlists.Playlist {
	stored, _ := state.Read[playlists.Stored](state.Playlists)
	return stored.Tracks
}

func (m *migration) storedFollows() follows.Section {
	section, _ := state.Read[follows.Section](state.Follows)
	return section
}

func (m *migration) storedSaved() saved.Section {
	section, _ := state.Read[saved.Section](state.Saved)
	return section
}

func (m *migration) storedOrder() ordering.Section {
	section, _ := state.Read[ordering.Section](state.Order)
	return section
}

func (m *migration) storedRooms() []rooms.Room {
	list, _ := state.Read[[]rooms.Room](state.Rooms)
	return list
}

func (m *migration) storedSessions() map[string]sessionEntry {
	sessions, _ := state.Read[map[string]sessionEntry](state.Sessions)
	return sessions
}

// spotifySession is the credentials of the Spotify session. Its own type lives
// in the spotify package, unexported, so the shape is mirrored here.
type spotifySession struct {
	Username string `json:"username"`
	Data     string `json:"data"`
}

// spotifyAccount is what the device keeps of Spotify: the credentials and the
// id that identifies this install to Connect.
type spotifyAccount struct {
	Session *spotifySession `json:"session,omitempty"`
	Device  string          `json:"device,omitempty"`
}

// youtubeAccount is what the device keeps of YouTube: the session and the video
// each song plays.
type youtubeAccount struct {
	Cookies string            `json:"cookies,omitempty"`
	Account string            `json:"account,omitempty"`
	Links   map[string]string `json:"links,omitempty"`
}

func (m *migration) storedSpotify() int {
	account, _ := state.Read[spotifyAccount](state.Spotify)
	if account.Session == nil {
		return 0
	}
	return 1
}

func (m *migration) storedYoutube() youtubeAccount {
	account, _ := state.Read[youtubeAccount](state.YouTube)
	return account
}

// value reads one key of the old store.
func (m *migration) value(bucket, key string) ([]byte, bool) {
	b, ok := m.buckets[bucket]
	if !ok {
		return nil, false
	}
	data, ok := b[key]
	return data, ok
}

// text reads a value the old store kept as plain text: some of them were
// written as a JSON string and some were not.
func text(data []byte) string {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		return s
	}
	return strings.TrimSpace(string(data))
}

// write stores a section, counting what was carried over.
func (m *migration) write(name string, value any, carried int) {
	if carried == 0 {
		return
	}
	if err := state.Write(name, value); err != nil {
		log.Printf("migrate: %s: %v", name, err)
		return
	}
	m.count += carried
}

// playlists: index + p:<id> + graves -> the playlists section.
func (m *migration) playlists() {
	legacy, ok := m.buckets["playlists"]
	if !ok {
		return
	}
	stored, _ := state.Read[playlists.Stored](state.Playlists)
	if stored.Tracks == nil {
		stored.Tracks = map[string]playlists.Playlist{}
	}
	before := make(map[string]bool, len(stored.Tracks))
	for id := range stored.Tracks {
		before[id] = true
	}

	source := map[string]bool{}
	carried := 0
	for key, data := range legacy {
		if !strings.HasPrefix(key, "p:") {
			continue
		}
		var playlist playlists.Playlist
		if err := json.Unmarshal(data, &playlist); err != nil || playlist.Id == "" {
			continue
		}
		source[playlist.Id] = true
		if _, ok := stored.Tracks[playlist.Id]; ok {
			continue
		}
		// A playlist this device removed stays removed: the store is another
		// device, and it still holds the copy that was dropped. Importing it
		// would undo the removal for good, since the store is deleted after.
		if _, ok := stored.Deleted[playlist.Id]; ok {
			continue
		}
		stored.Tracks[playlist.Id] = playlist
		carried++
	}

	// The index is the order of the shelves; a playlist the store knew and the
	// index does not is added at the end, so nothing is left out of the library.
	var legacyIndex []string
	if data, ok := legacy["index"]; ok {
		_ = json.Unmarshal(data, &legacyIndex)
	}
	index := append([]string{}, stored.Index...)
	seen := make(map[string]bool, len(index))
	for _, id := range index {
		seen[id] = true
	}
	for _, id := range legacyIndex {
		if seen[id] {
			continue
		}
		if _, ok := stored.Deleted[id]; ok {
			continue
		}
		seen[id] = true
		index = append(index, id)
		carried++
	}
	stored.Index = index

	if data, ok := legacy["graves"]; ok {
		var graves map[string]int64
		if err := json.Unmarshal(data, &graves); err == nil {
			if stored.Deleted == nil {
				stored.Deleted = map[string]int64{}
			}
			for id, at := range graves {
				if _, ok := stored.Deleted[id]; !ok {
					stored.Deleted[id] = at
					carried++
				}
			}
		}
	}

	kept, collapsed := collapse(&stored, source, before)
	m.expect["playlists"] = kept
	m.write(state.Playlists, stored, carried+collapsed)
}

// collapse leaves one local playlist per name. A local id is generated, so the
// same playlist reaches the library twice under two ids: the store's index
// keeps a copy the document already dropped, and an import of it adds that copy
// next to the one the user has. The newest copy wins, and the ones that lose
// are tombstoned so the backup does not bring them back.
//
// Only names the store knows are touched: a duplicate made in this version is
// the user's to keep. It answers how many playlists the store contributed and
// how many records changed.
func collapse(stored *playlists.Stored, source, before map[string]bool) (kept, changed int) {
	groups := map[string][]string{}
	for id, playlist := range stored.Tracks {
		// Only the local ones: a source-owned playlist is its own record and a
		// local one that happens to share its name is not a copy of it.
		if !strings.HasPrefix(id, music.LocalPrefix) {
			continue
		}
		if playlist.Name == "" {
			continue
		}
		groups[playlist.Name] = append(groups[playlist.Name], id)
	}
	for _, ids := range groups {
		fromStore := false
		for _, id := range ids {
			if source[id] {
				fromStore = true
				break
			}
		}
		if !fromStore {
			continue
		}
		kept++
		if len(ids) < 2 {
			continue
		}

		// The newest content wins; on a tie the smallest id does, so the answer
		// does not depend on the order the map is walked in.
		winner := ids[0]
		for _, id := range ids[1:] {
			if stored.Tracks[id].ModifiedAt < stored.Tracks[winner].ModifiedAt {
				continue
			}
			if stored.Tracks[id].ModifiedAt == stored.Tracks[winner].ModifiedAt && id > winner {
				continue
			}
			winner = id
		}

		// The id the library already used survives, so the shelf order and the
		// tombstones still point at it.
		keeper := winner
		if !before[winner] {
			for _, id := range ids {
				if before[id] && (keeper == winner || id < keeper) {
					keeper = id
				}
			}
		}

		playlist := stored.Tracks[winner]
		playlist.Id = keeper
		stored.Tracks[keeper] = playlist
		for _, id := range ids {
			if id == keeper {
				continue
			}
			delete(stored.Tracks, id)
			if slices.Contains(stored.Index, id) {
				index := make([]string, 0, len(stored.Index))
				for _, other := range stored.Index {
					if other != id {
						index = append(index, other)
					}
				}
				stored.Index = index
			}
			if stored.Deleted == nil {
				stored.Deleted = map[string]int64{}
			}
			if _, ok := stored.Deleted[id]; !ok {
				stored.Deleted[id] = time.Now().UnixMilli()
			}
			changed++
		}
	}
	return kept, changed
}

// follows: one key per account -> the follows section.
func (m *migration) follows() {
	legacy, ok := m.buckets["follows"]
	if !ok {
		return
	}
	section, _ := state.Read[follows.Section](state.Follows)
	items := make(map[string]int64, len(section.Items))
	for _, item := range section.Items {
		items[item.Id] = item.At
	}
	carried := map[string]bool{}
	for _, data := range legacy {
		for _, item := range followEntries(data) {
			if item.Id == "" {
				continue
			}
			if at, ok := items[item.Id]; ok && at >= item.At {
				continue
			}
			// Unfollowed on this device: the store still holds it, and bringing
			// it back would undo the removal for good.
			if at, ok := section.Removed[item.Id]; ok && at >= item.At {
				continue
			}
			// Only what the import carries is checked on the way back: a record
			// the device already had, or one it removed, is not missing.
			m.expect["follows"]++
			items[item.Id] = item.At
			carried[item.Id] = true
		}
	}
	if len(carried) == 0 {
		return
	}
	section.Items = nil
	for id, at := range items {
		section.Items = append(section.Items, follows.Entry{Id: id, At: at})
	}
	m.write(state.Follows, section, len(carried))
}

// followEntries reads the followed users of one account. The old store wrote
// them inside an object and, before that, as a bare list.
func followEntries(data []byte) []follows.Entry {
	var boxed struct {
		Items []follows.Entry `json:"items"`
	}
	if err := json.Unmarshal(data, &boxed); err == nil && boxed.Items != nil {
		return boxed.Items
	}
	var list []follows.Entry
	if err := json.Unmarshal(data, &list); err == nil {
		return list
	}
	return nil
}

// saved: index -> the saved section.
func (m *migration) saved() {
	var old saved.Section
	found := false
	// The store kept the list under "index" and, before that, under "list".
	for _, key := range []string{"index", "list"} {
		data, ok := m.value("saved-playlists", key)
		if !ok {
			continue
		}
		var part saved.Section
		if err := json.Unmarshal(data, &part); err != nil {
			continue
		}
		found = true
		old.Items = append(old.Items, part.Items...)
		for id, at := range part.Removed {
			if old.Removed == nil {
				old.Removed = map[string]int64{}
			}
			old.Removed[id] = at
		}
	}
	if !found {
		return
	}
	m.expect["saved"] = len(old.Items) + len(old.Removed)
	section, _ := state.Read[saved.Section](state.Saved)
	known := make(map[string]saved.Entry, len(section.Items))
	for _, item := range section.Items {
		known[item.Id] = item
	}
	carried := 0
	for _, item := range old.Items {
		if item.Id == "" {
			continue
		}
		if _, ok := known[item.Id]; ok {
			continue
		}
		// Unsaved on this device: the store still holds it, and bringing it
		// back would undo the removal for good.
		if at, ok := section.Removed[item.Id]; ok && at >= item.At {
			continue
		}
		known[item.Id] = item
		section.Items = append(section.Items, item)
		carried++
	}
	// The tombstones merge like any other record: the newest one wins, so a
	// removal the store knew is not lost because the device already had one.
	for id, at := range old.Removed {
		if section.Removed != nil {
			if current, ok := section.Removed[id]; ok && current >= at {
				continue
			}
		}
		if section.Removed == nil {
			section.Removed = map[string]int64{}
		}
		section.Removed[id] = at
		carried++
	}
	m.write(state.Saved, section, carried)
}

// order: one key per shelf -> the order section.
func (m *migration) order() {
	legacy, ok := m.buckets["ordering"]
	if !ok {
		return
	}
	section, _ := state.Read[ordering.Section](state.Order)
	shelves := make(map[string]ordering.Record, len(section))
	for _, entry := range section {
		shelves[shelfKey(entry.Scope, entry.Account)] = ordering.Record{
			Scope:   entry.Scope,
			Account: entry.Account,
			Ids:     entry.Ids,
			At:      entry.At,
		}
	}
	carried := map[string]bool{}
	for key, data := range legacy {
		record, ok := orderRecord(key, data)
		if !ok {
			continue
		}
		name := shelfKey(record.Scope, record.Account)
		if current, ok := shelves[name]; ok && current.At >= record.At {
			continue
		}
		shelves[name] = record
		carried[name] = true
	}
	// The old store kept a shelf twice (the list of an account before the shelf
	// and the record that replaced it), so what is expected is the number of
	// shelves and not the number of keys. The count is a set for the same
	// reason: whichever of the two the map yields first, the shelf is one.
	m.expect["order"] = len(shelves)
	if len(carried) == 0 {
		return
	}
	section = nil
	for _, record := range shelves {
		section = append(section, ordering.Entry{
			Scope:   record.Scope,
			Account: record.Account,
			Ids:     record.Ids,
			At:      record.At,
		})
	}
	m.write(state.Order, section, len(carried))
}

// shelfKey names one shelf: the scope alone when the shelf belongs to the app,
// the scope plus the account when it belongs to one.
func shelfKey(scope, account string) string {
	return scope + "\x00" + account
}

// orderRecord reads one shelf of the old store. The old format kept the
// library order of an account under its name; the new one names the shelf and
// the account apart.
func orderRecord(key string, data []byte) (ordering.Record, bool) {
	var record struct {
		Scope   string   `json:"scope"`
		Account string   `json:"account"`
		Ids     []string `json:"ids"`
		At      int64    `json:"at"`
	}
	if err := json.Unmarshal(data, &record); err == nil && len(record.Ids) > 0 {
		return ordering.Record{Scope: record.Scope, Account: record.Account, Ids: record.Ids, At: record.At}, true
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil || len(ids) == 0 {
		return ordering.Record{}, false
	}
	account := strings.TrimPrefix(key, "scope:")
	return ordering.Record{Scope: "library", Account: account, Ids: ids}, true
}

// rooms: saved -> the rooms section.
func (m *migration) rooms() {
	data, ok := m.value("rooms", "saved")
	if !ok {
		return
	}
	var old []rooms.Room
	if err := json.Unmarshal(data, &old); err != nil {
		return
	}
	current, _ := state.Read[[]rooms.Room](state.Rooms)
	known := make(map[string]bool, len(current))
	for _, room := range current {
		known[room.Id] = true
	}
	m.expect["rooms"] = len(old)
	carried := 0
	for _, room := range old {
		if room.Id == "" || known[room.Id] {
			continue
		}
		known[room.Id] = true
		current = append(current, room)
		carried++
	}
	m.write(state.Rooms, current, carried)
}

// settings: relay, session-pass, discord, idle-reload, sync -> the settings
// section.
func (m *migration) settings() {
	legacy, ok := m.buckets["settings"]
	if !ok {
		return
	}
	current, _ := state.Read[deviceSettings](state.Settings)
	// Read before anything is carried: the switches below are only taken on a
	// device that had no settings at all, and relay and pass are carried right
	// here, so the test has to answer what the device held, not what this
	// function has just written into current.
	fresh := current.Relay.URL == "" && current.SessionPass == ""
	carried := 0

	if current.Relay.URL == "" {
		if data, ok := legacy["relay"]; ok {
			var relay Relay
			if err := json.Unmarshal(data, &relay); err == nil && relay.URL != "" {
				current.Relay = relay
				current.RelayAt = time.Now().UnixMilli()
				carried++
			}
		}
	}
	if current.SessionPass == "" {
		// The pass has had two names: the settings that carried the YouTube
		// cookies used the older one, and a device that kept it is the one whose
		// sealed sessions would stop opening.
		if data, ok := legacy["session-pass"]; ok {
			var pass string
			if err := json.Unmarshal(data, &pass); err == nil && pass != "" {
				current.SessionPass = pass
				carried++
			}
		}
		if current.SessionPass == "" {
			if data, ok := legacy["youtube-cookie-pass"]; ok {
				var pass string
				if err := json.Unmarshal(data, &pass); err == nil && pass != "" {
					current.SessionPass = pass
					carried++
				}
			}
		}
	}
	// The folder the document lives in belongs to the device, but the user
	// picked it once and would have to pick it again.
	if current.Sync.Folder == "" {
		if data, ok := legacy["sync-folder"]; ok {
			var folder string
			if err := json.Unmarshal(data, &folder); err == nil && folder != "" {
				current.Sync.Folder = folder
				carried++
			}
		}
	}
	if current.Sync.Interval == nil {
		if data, ok := legacy["sync-interval"]; ok {
			var interval int
			if err := json.Unmarshal(data, &interval); err == nil && interval > 0 {
				current.Sync.Interval = &interval
				carried++
			}
		}
	}
	// The two switches decide whether a pass applies the document and whether
	// the folder is reconciled at all. A device that turned one off would have
	// it back on: a missing switch means on, so it has to be carried.
	if current.Sync.Import == nil {
		if data, ok := legacy["sync-import"]; ok {
			var imported bool
			if err := json.Unmarshal(data, &imported); err == nil {
				current.Sync.Import = &imported
				carried++
			}
		}
	}
	if current.Sync.Auto == nil {
		if data, ok := legacy["sync-auto"]; ok {
			var auto bool
			if err := json.Unmarshal(data, &auto); err == nil {
				current.Sync.Auto = &auto
				carried++
			}
		}
	}
	// The two switches are only carried when the device holds no settings at
	// all: a switch the user turned off after the upgrade must stay off.
	if fresh {
		if data, ok := legacy["discord"]; ok {
			var discord bool
			if err := json.Unmarshal(data, &discord); err == nil && discord {
				current.Discord = true
				carried++
			}
		}
		if data, ok := legacy["idle-reload"]; ok {
			var idle bool
			if err := json.Unmarshal(data, &idle); err == nil && idle {
				on := true
				current.IdleReload = &on
				carried++
			}
		}
	}

	m.write(state.Settings, current, carried)
}

// sessions: the encrypted blobs -> the sessions section.
func (m *migration) sessions() {
	legacy, ok := m.buckets["settings"]
	if !ok {
		return
	}
	current, _ := state.Read[map[string]sessionEntry](state.Sessions)
	if current == nil {
		current = map[string]sessionEntry{}
	}
	carried := 0
	for _, source := range []string{"spotify", "youtube"} {
		if _, ok := current[source]; ok {
			continue
		}
		data, ok := legacy["session:"+source]
		if !ok {
			continue
		}
		var blob string
		if err := json.Unmarshal(data, &blob); err != nil || blob == "" {
			continue
		}
		current[source] = sessionEntry{Value: blob, At: time.Now().UnixMilli()}
		m.expect["sessions"]++
		carried++
	}
	m.write(state.Sessions, current, carried)
}

// spotify: the credentials and the device id -> the spotify section.
func (m *migration) spotify() {
	current, _ := state.Read[spotifyAccount](state.Spotify)
	carried := 0

	if current.Session == nil {
		if data, ok := m.value("spotify-global", "global"); ok {
			var session spotifySession
			if err := json.Unmarshal(data, &session); err == nil && session.Username != "" {
				current.Session = &session
				m.expect["spotify"] = 1
				carried++
			}
		}
	}
	if current.Device == "" {
		if data, ok := m.value("spotify-device", "device-id"); ok {
			if device := text(data); device != "" {
				current.Device = device
				carried++
			}
		}
	}

	m.write(state.Spotify, current, carried)
}

// youtube: the cookies, the account and the links -> the youtube section.
func (m *migration) youtube() {
	current, _ := state.Read[youtubeAccount](state.YouTube)
	carried := 0

	if current.Cookies == "" {
		if data, ok := m.value("youtube-auth", "auth"); ok {
			var cookies string
			if err := json.Unmarshal(data, &cookies); err == nil && cookies != "" {
				current.Cookies = cookies
				carried++
			}
		}
	}
	if current.Account == "" {
		if data, ok := m.value("youtube-auth", "account"); ok {
			var account string
			if err := json.Unmarshal(data, &account); err == nil && account != "" {
				current.Account = account
				carried++
			}
		}
	}
	// The links are the video each song plays: a pick the user made, so they
	// travel with the state. The old store named a Spotify song "spotify:<id>";
	// the new one says which kind of song it is.
	//
	// They merge by song and not only when the device has none: a device that
	// has played a handful of songs since the upgrade must still take the rest.
	// A link the device already holds is the newer pick, so it stays.
	if sources, ok := m.buckets["youtube-source"]; ok {
		links := map[string]string{}
		for key, data := range sources {
			song, video := link(key, data)
			if song == "" || video == "" {
				continue
			}
			links[song] = video
		}
		m.expect["youtube"] = len(links)
		if current.Links == nil {
			current.Links = map[string]string{}
		}
		for song, video := range links {
			if _, ok := current.Links[song]; ok {
				continue
			}
			current.Links[song] = video
			carried++
		}
	}

	m.write(state.YouTube, current, carried)
}

// link reads one song and the video it plays from the old store. The old keys
// were the song id and a bare hash, and only the Spotify ones still name the
// song the new store knows.
func link(key string, data []byte) (string, string) {
	if strings.HasPrefix(key, "spotify:") && !strings.HasPrefix(key, "spotify:track:") &&
		!strings.HasPrefix(key, "spotify:album:") && !strings.HasPrefix(key, "spotify:artist:") {
		key = "spotify:track:" + strings.TrimPrefix(key, "spotify:")
	}
	if !strings.HasPrefix(key, "spotify:track:") {
		return "", ""
	}
	return key, strings.TrimPrefix(text(data), "youtube:")
}

// window: state -> window.json, next to the log.
func (m *migration) window() {
	data, ok := m.value("window", "state")
	if !ok {
		return
	}
	path := filepath.Join(filepath.Dir(LogPath()), "window.json")
	// The app writes this file on its own. An import that is retried must not
	// move a window the user already placed, and the write is not atomic.
	if _, err := os.Stat(path); err == nil {
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Printf("migrate: window: %v", err)
		return
	}
	m.count++
}
