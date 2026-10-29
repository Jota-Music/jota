package kv

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
)

func TestReopenAfterClose(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	Close()

	if err := EnsureStarted(); err != nil {
		t.Fatalf("open: %v", err)
	}
	if db == nil {
		t.Fatal("db is nil after open")
	}

	Close()
	if db != nil {
		t.Fatal("db not nil after close")
	}

	if err := EnsureStarted(); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if db == nil {
		t.Fatal("db is nil after reopen")
	}
	Close()
}

// The window opens before the previous instance has released the store, so a
// lock is a handoff rather than a reason to run the whole session without
// persistence.
func TestStartRetriesWhileLocked(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	// Start is a first-launch call: the previous test's Close would veto it.
	mu.Lock()
	openErr = nil
	mu.Unlock()
	t.Cleanup(Close)

	path := filepath.Join(dir, "jota", "storage", "kv")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	held, err := badger.Open(badger.DefaultOptions(path).WithLogger(nil))
	if err != nil {
		t.Fatalf("hold the store: %v", err)
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = held.Close()
	}()

	Start()
	if db == nil {
		t.Fatal("start gave up while the previous instance was still releasing the store")
	}
}
