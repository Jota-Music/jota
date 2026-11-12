package legacy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRead reads a store written by Badger: the manifest, the table and the
// values are the ones a real store holds.
func TestRead(t *testing.T) {
	store, skipped, err := Read("../app/testdata/store")
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Fatalf("skipped %d values", skipped)
	}
	settings := store["settings"]
	if settings == nil {
		t.Fatal("no settings bucket")
	}
	var relay map[string]string
	if err := json.Unmarshal(settings["relay"], &relay); err != nil {
		t.Fatal(err)
	}
	if relay["url"] != "relay.example" || relay["token"] != "tok" {
		t.Fatalf("relay: %v", relay)
	}
	// A key written twice is alive once, with the newest value.
	if got := string(settings["discord"]); got != "false" {
		t.Fatalf("discord: %s", got)
	}
	// A deleted key is gone.
	if _, ok := settings["gone"]; ok {
		t.Fatal("the deleted key is still there")
	}
	// A value the store kept as plain text is read as it is.
	if got := string(store["spotify-device"]["device-id"]); got != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("device: %s", got)
	}
}

// TestExpired checks the TTL an entry may carry, without a fixture that would
// only expire a second after it is written.
func TestExpired(t *testing.T) {
	now := time.Unix(1700000000, 0)
	if (entry{}).expired(now) {
		t.Fatal("an entry without a TTL expired")
	}
	if !(entry{expiresAt: 1699999999}).expired(now) {
		t.Fatal("a passed TTL did not expire")
	}
	if (entry{expiresAt: 1700000001}).expired(now) {
		t.Fatal("a live TTL expired")
	}
}

// TestReadWithoutStore says what is wrong instead of failing blind.
func TestReadWithoutStore(t *testing.T) {
	if _, _, err := Read(t.TempDir()); err == nil {
		t.Fatal("a directory without a manifest was read")
	}
}

// TestReadRefusesAnotherVersion checks that a store of another format is
// refused instead of half read.
func TestReadRefusesAnotherVersion(t *testing.T) {
	dir := t.TempDir()
	raw := []byte("Bdgr\x00\x00\x00\x09")
	if err := os.WriteFile(filepath.Join(dir, "MANIFEST"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Read(dir); err == nil {
		t.Fatal("a store of another version was read")
	}
}
