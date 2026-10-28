package kv

import "testing"

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

// The startup open runs in the background. A shutdown that lands first must not
// leave a database open that nothing will close, and an explicit later use must
// still be able to open one.
func TestStartAfterCloseDoesNotReopen(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)

	Close()
	Start()
	if db != nil {
		t.Fatal("a start that raced the shutdown left the database open")
	}

	t.Cleanup(Close)
	if err := EnsureStarted(); err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if db == nil {
		t.Fatal("db is nil after an explicit open")
	}
}
