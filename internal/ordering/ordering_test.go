package ordering

import (
	"reflect"
	"testing"

	"github.com/Jota-Music/jota/internal/kv"
)

func TestOrderLifecycle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)

	s := New()

	if ids, err := s.List("alice"); err != nil || len(ids) != 0 {
		t.Fatalf("empty list = %v, %v", ids, err)
	}

	if err := s.Save("Alice", Order{"b", "a", "b", "  ", "c"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := s.List("ALICE")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if want := (Order{"b", "a", "c"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v, want %v", got, want)
	}

	other, err := s.List("carol")
	if err != nil {
		t.Fatalf("list other: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("other account leaked order: %v", other)
	}
}
