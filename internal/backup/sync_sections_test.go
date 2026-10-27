package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Jota-Music/jota/internal/follows"
	"github.com/Jota-Music/jota/internal/music"
	"github.com/Jota-Music/jota/internal/ordering"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/Jota-Music/jota/internal/services/discord/saved"
)

func device(settings SettingsStore) *Service {
	return &Service{
		Playlists: playlists.New(nil),
		Follows:   follows.New(),
		Saved:     saved.New(),
		Order:     ordering.New(),
		Settings:  settings,
	}
}

// A document exported on one device brings every section back on another, and a
// pass with import off leaves the library alone while still writing the union.
func TestSyncRestoresEverySection(t *testing.T) {
	setup(t)
	relay := Setting{Value: json.RawMessage(`{"URL":"r","Token":"t"}`), At: 10}
	source := device(fakeSettings{})
	if _, err := source.Playlists.Create("Road trip"); err != nil {
		t.Fatal(err)
	}
	if err := source.Follows.Follow("friend"); err != nil {
		t.Fatal(err)
	}
	if err := source.Saved.Add(music.PlaylistSummary{Id: "spotify:playlist:x", Name: "Kept"}); err != nil {
		t.Fatal(err)
	}
	if err := source.Order.Save(ordering.Library, "me@example.com", []string{"b", "a"}); err != nil {
		t.Fatal(err)
	}
	text, err := source.Export(nil)
	if err != nil {
		t.Fatal(err)
	}
	var doc Document
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatal(err)
	}
	doc.Settings = Settings{"relay": relay}
	text, err = marshal(doc)
	if err != nil {
		t.Fatal(err)
	}

	setup(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, BackupFile), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	target := device(fakeSettings{})
	if _, err := target.Sync(dir, Options{Import: true}); err != nil {
		t.Fatal(err)
	}

	list, _ := target.Playlists.List()
	if len(list) != 1 || list[0].Name != "Road trip" {
		t.Fatalf("playlists = %+v", list)
	}
	users, _ := target.Follows.List()
	if len(users) != 1 || users[0] != "friend" {
		t.Fatalf("follows = %v", users)
	}
	kept, _ := target.Saved.List()
	if len(kept) != 1 || kept[0].Id != "spotify:playlist:x" {
		t.Fatalf("saved = %+v", kept)
	}
	order, _ := target.Order.List(ordering.Library, "me@example.com")
	if len(order) != 2 || order[0] != "b" {
		t.Fatalf("order = %v", order)
	}

	setup(t)
	quiet := device(fakeSettings{})
	own, err := quiet.Playlists.Create("Mine")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := quiet.Sync(dir, Options{}); err != nil {
		t.Fatal(err)
	}
	mine, _ := quiet.Playlists.List()
	if len(mine) != 1 || mine[0].Id != own.Id {
		t.Fatalf("library changed without import: %+v", mine)
	}
	users, _ = quiet.Follows.List()
	if len(users) != 0 {
		t.Fatalf("follows changed without import: %v", users)
	}
	if got := ids(readDoc(t, filepath.Join(dir, BackupFile))); len(got) != 2 {
		t.Fatalf("document = %v, want the union", got)
	}
}
