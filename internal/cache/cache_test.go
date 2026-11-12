package cache

import (
	"testing"
	"time"
)

func setup(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
}

func TestGetWithoutAnEntryIsEmpty(t *testing.T) {
	setup(t)

	if _, ok := Get[string]("playlists/one"); ok {
		t.Fatal("an entry that was never written exists")
	}
}

func TestPutAndGetBack(t *testing.T) {
	setup(t)

	if err := Put("playlists/one", "a playlist"); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, ok := Get[string]("playlists/one")
	if !ok || got != "a playlist" {
		t.Fatalf("entry = %q, %v", got, ok)
	}
}

// The cache stores one row per entity under the name it was given.
func TestTheEntryIsStoredUnderItsName(t *testing.T) {
	setup(t)

	if err := Put("playlists/5zYozysNnBKny8EhXzgVL7", "a playlist"); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, ok := Get[string]("playlists/5zYozysNnBKny8EhXzgVL7")
	if !ok || got != "a playlist" {
		t.Fatalf("entry = %q, %v", got, ok)
	}
}

// An entry that expired is not there: the source answers it again.
func TestAnExpiredEntryIsGone(t *testing.T) {
	setup(t)

	if err := Put("audio/one", "a stream", 40*time.Millisecond); err != nil {
		t.Fatalf("put: %v", err)
	}
	time.Sleep(60 * time.Millisecond)

	if _, ok := Get[string]("audio/one"); ok {
		t.Fatal("an expired entry is still there")
	}
}

func TestForgetIsIdempotent(t *testing.T) {
	setup(t)

	if err := Put("playlists/one", "a playlist"); err != nil {
		t.Fatalf("put: %v", err)
	}
	for range 2 {
		if err := Forget("playlists/one"); err != nil {
			t.Fatalf("forget: %v", err)
		}
	}
}

// The refresh walks the cache to find what it has to revalidate.
func TestWalkVisitsWhatWasStored(t *testing.T) {
	setup(t)

	if err := Put("playlists/one", "a"); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := Put("playlists/two", "b"); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := Put("tracks/three", "c"); err != nil {
		t.Fatalf("put: %v", err)
	}

	seen := map[string]bool{}
	if err := Walk("playlists", func(name string) error {
		seen[name] = true
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	if !seen["playlists/one"] || !seen["playlists/two"] {
		t.Fatalf("walked %v", seen)
	}
	if seen["tracks/three"] {
		t.Fatal("the walk left its directory")
	}
}
