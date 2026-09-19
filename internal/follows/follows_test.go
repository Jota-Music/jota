package follows

import (
	"reflect"
	"testing"

	"github.com/Jota-Music/jota/internal/kv"
)

func TestFollowLifecycle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)

	s := New()

	if err := s.Follow("Alice", "Bob"); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := s.Follow("alice", "bob"); err != nil {
		t.Fatalf("duplicate follow: %v", err)
	}

	got, err := s.List("ALICE")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if want := (Users{"Bob"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("list = %v, want %v", got, want)
	}

	other, err := s.List("carol")
	if err != nil {
		t.Fatalf("list other: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("other account leaked follows: %v", other)
	}

	if err := s.Unfollow("alice", "BOB"); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	got, err = s.List("alice")
	if err != nil {
		t.Fatalf("list after unfollow: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("after unfollow = %v, want empty", got)
	}
}
