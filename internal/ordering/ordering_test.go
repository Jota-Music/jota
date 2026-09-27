package ordering

import (
	"reflect"
	"testing"

	"github.com/Jota-Music/jota/internal/kv"
)

func setup(t *testing.T) *Service {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)
	return New()
}

func TestOrderLifecycle(t *testing.T) {
	s := setup(t)

	if ids, err := s.List(Library, "alice"); err != nil || len(ids) != 0 {
		t.Fatalf("empty list = %v, %v", ids, err)
	}

	if err := s.Save(Library, "Alice", Order{"b", "a", "b", "  ", "c"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.List(Library, "ALICE")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if want := (Order{"b", "a", "c"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v, want %v", got, want)
	}

	other, err := s.List(Library, "carol")
	if err != nil {
		t.Fatalf("list other: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("other account leaked order: %v", other)
	}
}

func TestScopesAreSeparate(t *testing.T) {
	s := setup(t)

	if err := s.Save(Library, "alice", Order{"a", "b"}); err != nil {
		t.Fatalf("save library: %v", err)
	}
	if err := s.Save("following", "alice", Order{"b", "a"}); err != nil {
		t.Fatalf("save following: %v", err)
	}

	library, _ := s.List(Library, "alice")
	if want := (Order{"a", "b"}); !reflect.DeepEqual(library, want) {
		t.Fatalf("library = %v, want %v", library, want)
	}
	following, _ := s.List("following", "alice")
	if want := (Order{"b", "a"}); !reflect.DeepEqual(following, want) {
		t.Fatalf("following = %v, want %v", following, want)
	}
}

func TestOrderWithoutAccount(t *testing.T) {
	s := setup(t)

	if err := s.Save("rooms", "", Order{"r1", "r2"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, _ := s.List("rooms", "")
	if want := (Order{"r1", "r2"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("rooms = %v, want %v", got, want)
	}
	if other, _ := s.List(Library, ""); len(other) != 0 {
		t.Fatalf("global order leaked into another scope: %v", other)
	}
}

// Orders were stored under the bare account key before shelves had scopes, so
// the library order has to keep reading it.
func TestLibraryReadsLegacyKey(t *testing.T) {
	s := setup(t)

	if err := bucket.SetObject("alice", Order{"x", "y"}); err != nil {
		t.Fatalf("legacy save: %v", err)
	}

	got, err := s.List(Library, "alice")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if want := (Order{"x", "y"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy library = %v, want %v", got, want)
	}

	if err := s.Save(Library, "alice", Order{"y", "x"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	after, _ := s.List(Library, "alice")
	if want := (Order{"y", "x"}); !reflect.DeepEqual(after, want) {
		t.Fatalf("scoped order = %v, want %v", after, want)
	}
	legacy, _ := read("alice")
	if want := (Order{"x", "y"}); !reflect.DeepEqual(legacy, want) {
		t.Fatalf("legacy key rewritten: %v, want %v", legacy, want)
	}
}
