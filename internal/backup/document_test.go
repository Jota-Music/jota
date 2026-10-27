package backup

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/Jota-Music/jota/internal/follows"
	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/ordering"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/services/discord/saved"
)

// empty returns a library with every section wired and nothing in it, the way a
// fresh profile starts.
func empty(t *testing.T) *Service {
	t.Helper()
	setup(t)
	return &Service{
		Playlists: playlists.New(func(id string) (music.Song, error) {
			return music.Song{Id: id, Name: "Song"}, nil
		}),
		Follows:  follows.New(),
		Saved:    saved.New(),
		Order:    ordering.New(),
		Settings: fakeSettings{},
	}
}

// seeded returns a library with one record in every section, the way a device in
// use has one.
func seeded(t *testing.T) *Service {
	t.Helper()
	s := empty(t)
	s.Settings = fakeSettings{"relay": {Value: json.RawMessage(`"https://relay"`), At: 10}}

	if err := s.Follows.Follow("bob"); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := s.Saved.Add(music.PlaylistSummary{Id: "spotify:p1", Name: "Liked"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := s.Order.Save(ordering.Library, "alice", ordering.Order{"a", "b"}); err != nil {
		t.Fatalf("order: %v", err)
	}
	summary, err := s.Playlists.Create("Mix")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := s.Playlists.AddSongs(summary.Id, []string{"spotify:track:a"}); err != nil {
		t.Fatalf("add songs: %v", err)
	}
	return s
}

// The document carries all five sections, and importing it into a fresh profile
// rebuilds every one of them.
func TestDocumentRoundTrip(t *testing.T) {
	snapshot := playlists.Snapshot{
		Id: "spotify:p2", Source: "spotify", Name: "Owned",
		Songs: []music.Song{{Id: "spotify:track:b", Name: "Remote"}},
	}
	text, err := seeded(t).Export([]playlists.Snapshot{snapshot})
	if err != nil {
		t.Fatalf("export: %v", err)
	}

	doc, err := Parse([]byte(text))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if doc.Version != Version {
		t.Fatalf("document = %+v, want a versioned one", doc)
	}
	if len(doc.Playlists.Items) != 1 || len(doc.Playlists.External) != 1 {
		t.Fatalf("playlists = %+v, want the local one and the source snapshot", doc.Playlists)
	}
	if len(doc.Follows.Items) != 1 {
		t.Fatalf("follows = %+v", doc.Follows)
	}
	if len(doc.Saved.Items) != 1 {
		t.Fatalf("saved = %+v", doc.Saved)
	}
	if len(doc.Order) != 1 {
		t.Fatalf("order = %+v", doc.Order)
	}
	if got := string(doc.Settings["relay"].Value); got != `"https://relay"` {
		t.Fatalf("settings = %s", got)
	}

	dst := empty(t)
	merged, stats, err := dst.ImportDocument(text, nil, false, true)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if stats.Playlists != 1 || stats.Follows != 1 || stats.Saved != 1 || stats.Order != 1 || stats.Settings != 1 {
		t.Fatalf("stats = %+v, want one record per section", stats)
	}
	if stats.Modified {
		t.Fatalf("importing a document this build wrote reported a change:\n%s\n%s", text, merged)
	}

	lists, err := dst.Playlists.List()
	if err != nil || len(lists) != 1 || lists[0].Name != "Mix" {
		t.Fatalf("playlists = %+v, %v", lists, err)
	}
	if users, _ := dst.Follows.List(); !reflect.DeepEqual(users, follows.Users{"bob"}) {
		t.Fatalf("follows = %v", users)
	}
	if kept, _ := dst.Saved.List(); len(kept) != 1 || kept[0].Id != "spotify:p1" {
		t.Fatalf("saved = %+v", kept)
	}
	if order, _ := dst.Order.List(ordering.Library, "alice"); !reflect.DeepEqual(order, ordering.Order{"a", "b"}) {
		t.Fatalf("order = %v", order)
	}
	if got := string(dst.Settings.(fakeSettings)["relay"].Value); got != `"https://relay"` {
		t.Fatalf("settings = %s", got)
	}
}

// A document of a version this build does not write is rejected instead of being
// guessed at, and a blank file is an empty document, not a broken one.
func TestDocumentRejectsOtherVersions(t *testing.T) {
	for _, text := range []string{`{"version":2}`, `{"version":0}`, `{"playlists":{"items":[]}}`} {
		if _, err := Parse([]byte(text)); !errors.Is(err, ErrVersion) {
			t.Fatalf("parsed %s: %v, want ErrVersion", text, err)
		}
	}

	// A copy a provider cut short is not another version: it is unreadable, and
	// the panel has to say so instead of sending the user after a version.
	if _, err := Parse([]byte(`{"version":1,"playlists":{"items":[`)); !errors.Is(err, ErrUnreadable) {
		t.Fatalf("truncated document = %v, want ErrUnreadable", err)
	}

	doc, err := Parse([]byte(" \n"))
	if err != nil || doc.Version != Version || !doc.Empty() {
		t.Fatalf("blank file = %+v, %v; want an empty document", doc, err)
	}
}

// A section nobody wrote stays out of the file, so an empty library does not
// write five empty objects.
func TestDocumentOmitsEmptySections(t *testing.T) {
	doc := Document{Version: Version}
	if !doc.Empty() {
		t.Fatal("empty document reported content")
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if want := `{"version":1}`; string(data) != want {
		t.Fatalf("document = %s, want %s", data, want)
	}

	doc.Follows = follows.Section{Items: []follows.Entry{{Id: "bob", At: 3}}}
	data, err = json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var present map[string]json.RawMessage
	if err := json.Unmarshal(data, &present); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := present["follows"]; !ok {
		t.Fatalf("follows section missing: %s", data)
	}
	for _, key := range []string{"playlists", "saved", "order", "settings"} {
		if _, ok := present[key]; ok {
			t.Fatalf("%s written although empty: %s", key, data)
		}
	}
}

// The playlists Spotify and YouTube own travel in the file so another profile
// shows them, but an import never recreates them as local playlists.
func TestDocumentKeepsSourcePlaylistsOutOfStore(t *testing.T) {
	text := `{"version":1,"playlists":{"items":[` +
		`{"id":"local:mine","source":"local","name":"Mine","songs":[]},` +
		`{"id":"spotify:p1","source":"spotify","name":"Liked","songs":[]},` +
		`{"id":"youtube:p2","source":"youtube","name":"Watch later","songs":[]}]}}`

	s := empty(t)
	merged, stats, err := s.ImportDocument(text, nil, false, true)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if stats.Playlists != 1 {
		t.Fatalf("imported = %d, want 1 local playlist", stats.Playlists)
	}
	if list, _ := s.Playlists.List(); len(list) != 1 || list[0].Name != "Mine" {
		t.Fatalf("store = %+v, want only the local playlist", list)
	}

	// They are still readable, so the next device sees them in its shelf.
	doc, err := Parse([]byte(merged))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(doc.Playlists.Items) != 1 || len(doc.Playlists.External) != 2 {
		t.Fatalf("playlists = %+v, want the local one and both snapshots", doc.Playlists)
	}
	names := map[string]string{}
	for _, ext := range doc.Playlists.External {
		names[ext.Source] = ext.Name
	}
	if names["spotify"] != "Liked" || names["youtube"] != "Watch later" {
		t.Fatalf("snapshots = %+v", doc.Playlists.External)
	}
}

// Shelves of both shapes travel: an app-wide one like rooms and one owned by an
// account. A newer list wins a merge, and an explicit restore lets the file win
// even when its list is older.
func TestDocumentCarriesShelfOrders(t *testing.T) {
	setup(t)
	source := empty(t)
	if err := source.Order.Save("rooms", "", ordering.Order{"r1", "r2"}); err != nil {
		t.Fatalf("save rooms: %v", err)
	}
	if err := source.Order.Save(ordering.Library, "me@example.com", ordering.Order{"a", "b"}); err != nil {
		t.Fatalf("save library: %v", err)
	}
	local, err := source.Order.Section()
	if err != nil {
		t.Fatalf("section: %v", err)
	}
	roomsAt, libraryAt := int64(0), int64(0)
	for _, cur := range local {
		switch cur.Scope {
		case "rooms":
			roomsAt = cur.At
		case ordering.Library:
			libraryAt = cur.At
		}
	}

	text, err := source.Export(nil)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	setup(t)
	target := empty(t)
	if _, _, err := target.ImportDocument(text, nil, false, true); err != nil {
		t.Fatalf("import: %v", err)
	}
	if got, _ := target.Order.List("rooms", ""); !reflect.DeepEqual(got, ordering.Order{"r1", "r2"}) {
		t.Fatalf("rooms = %v", got)
	}
	if got, _ := target.Order.List(ordering.Library, "me@example.com"); !reflect.DeepEqual(got, ordering.Order{"a", "b"}) {
		t.Fatalf("library = %v", got)
	}

	newer := Document{Version: Version, Order: ordering.Section{
		{Scope: ordering.Library, Account: "me@example.com", Ids: []string{"x"}, At: libraryAt + 1},
		{Scope: "rooms", Ids: []string{"old"}, At: roomsAt - 1},
	}}
	if _, stats, err := target.Merge(newer, true); err != nil {
		t.Fatalf("merge: %v", err)
	} else if stats.Order != 1 {
		t.Fatalf("merged %d shelves, want only the newer one", stats.Order)
	}
	if got, _ := target.Order.List(ordering.Library, "me@example.com"); !reflect.DeepEqual(got, ordering.Order{"x"}) {
		t.Fatalf("library = %v, want the newer list", got)
	}
	if got, _ := target.Order.List("rooms", ""); !reflect.DeepEqual(got, ordering.Order{"r1", "r2"}) {
		t.Fatalf("rooms = %v, want the newer local list", got)
	}

	restore := Document{Version: Version, Order: ordering.Section{
		{Scope: "rooms", Ids: []string{"forced"}, At: roomsAt - 2},
	}}
	restoreText, err := marshal(restore)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := target.Restore(restoreText); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if got, _ := target.Order.List("rooms", ""); !reflect.DeepEqual(got, ordering.Order{"forced"}) {
		t.Fatalf("rooms = %v, want the file's list after a restore", got)
	}
}
