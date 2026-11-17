package rooms

import (
	"testing"

	"github.com/Jota-Music/jota/internal/state"
)

func TestRoomLifecycle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	state.Reset()
	t.Cleanup(state.Reset)

	s := NewStore()

	saved, err := s.Save(Room{Code: " party ", RelayURL: "relay.example.com", Password: "pw"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if len(saved) != 1 {
		t.Fatalf("saved = %v", saved)
	}
	if saved[0].Name != "party" || saved[0].Id != "local:room:relay.example.com|party" {
		t.Fatalf("normalized = %+v", saved[0])
	}
	if saved[0].Password != "pw" {
		t.Fatalf("password not stored: %+v", saved[0])
	}

	// Same relay+code replaces instead of appending.
	saved, err = s.Save(Room{Code: "party", RelayURL: "relay.example.com", Name: "Jam"})
	if err != nil {
		t.Fatalf("resave: %v", err)
	}
	if len(saved) != 1 || saved[0].Name != "Jam" {
		t.Fatalf("after resave = %v", saved)
	}

	// A second relay makes a distinct room.
	if _, err := s.Save(Room{Code: "party", RelayURL: "other.example.com"}); err != nil {
		t.Fatalf("save other relay: %v", err)
	}

	// The relay is case-sensitive, so a differently-cased code is another room.
	if _, err := s.Save(Room{Code: "Party", RelayURL: "relay.example.com"}); err != nil {
		t.Fatalf("save cased code: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("list = %v", got)
	}

	left, err := s.Remove("local:room:relay.example.com|party")
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(left) != 2 {
		t.Fatalf("after remove = %v", left)
	}
	for _, r := range left {
		if r.Code == "party" && r.RelayURL == "relay.example.com" {
			t.Fatalf("removed the wrong room: %v", left)
		}
	}

	if _, err := s.Save(Room{Code: "no-relay"}); err == nil {
		t.Fatal("expected error for room without a relay")
	}
}

// A store written before the id rule keeps its rooms: the key the page gets is
// the composed one, and removing by it finds the bare record.
func TestRoomsSavedBeforeTheIdRuleStillAnswerTheCurrentKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	state.Reset()
	t.Cleanup(state.Reset)

	legacy := []Room{{
		Id:       "relay.example.com|party",
		Code:     "party",
		RelayURL: "relay.example.com",
		Password: "pw",
		Name:     "party",
		SavedAt:  1,
	}}
	if err := state.Write(state.Rooms, legacy); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	s := NewStore()
	got, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Id != "local:room:relay.example.com|party" {
		t.Fatalf("list = %+v", got)
	}

	left, err := s.Remove(got[0].Id)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(left) != 0 {
		t.Fatalf("after remove = %+v", left)
	}
}

func TestRoomSecretsPersist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	state.Reset()
	t.Cleanup(state.Reset)

	s := NewStore()
	if _, err := s.Save(Room{
		Code:     "locked",
		RelayURL: "relay.test",
		Token:    "tok",
		Password: "hunter2",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("list = %+v", got)
	}
	if got[0].Password != "hunter2" || got[0].Token != "tok" {
		t.Fatalf("secrets not persisted: %+v", got[0])
	}
}
