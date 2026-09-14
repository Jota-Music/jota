package kv

import "testing"

func TestReopenAfterClose(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)

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
