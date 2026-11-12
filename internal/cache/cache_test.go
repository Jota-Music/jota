package cache

import (
	"os"
	"path/filepath"
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

// The sweep collects the entries whose life is over: one that old cannot be
// alive any more, and a fresh one is left alone.
func TestSweepDropsOnlyWhatIsOld(t *testing.T) {
	setup(t)

	path, err := cachePath("streams/one")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dead := filepath.Join(filepath.Dir(path), "dead.json")
	if err := os.WriteFile(dead, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(dead, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	alive := filepath.Join(filepath.Dir(path), "alive.json")
	if err := os.WriteFile(alive, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if dropped := Sweep("streams", 24*time.Hour); dropped != 1 {
		t.Fatalf("dropped %d, want the one that is old", dropped)
	}
	if _, err := os.Stat(dead); !os.IsNotExist(err) {
		t.Fatalf("the dead entry is still there: %v", err)
	}
	if _, err := os.Stat(alive); err != nil {
		t.Fatalf("the fresh entry was collected: %v", err)
	}
}

// A write that died between the temporary file and the rename leaves one
// behind, and nothing else would ever collect it.
func TestSweepRemovesTheLeftoversOfACrashedWrite(t *testing.T) {
	setup(t)

	path, err := cachePath("playlists/one")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	leftover := path + ".tmp"
	if err := os.WriteFile(leftover, []byte("{half"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	// The write is older than the process, so no write of this run owns it.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(leftover, old, old); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	sweep()

	if _, err := os.Stat(leftover); !os.IsNotExist(err) {
		t.Fatalf("the leftover is still there: %v", err)
	}
}

// A write in flight owns its temporary file: the sweep leaves it alone.
func TestSweepLeavesAWriteInFlightAlone(t *testing.T) {
	setup(t)

	path, err := cachePath("playlists/one")
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	inFlight := path + ".tmp"
	if err := os.WriteFile(inFlight, []byte("half"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	sweep()

	if _, err := os.Stat(inFlight); err != nil {
		t.Fatalf("the write in flight was collected: %v", err)
	}
}
