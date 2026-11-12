package legacy

import (
	"encoding/binary"
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

// TestReadWAL reads a memtable log: the writes a store holds that were not
// flushed to a table yet, which is what a killed app leaves behind. A delete
// sits in the middle of it.
func TestReadWAL(t *testing.T) {
	store, skipped, err := Read("../app/testdata/wal")
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
	if got := string(settings["relay"]); got != `{"url":"relay.example","token":"tok"}` {
		t.Fatalf("relay: %s", got)
	}
	// The write that follows the delete is the newest one and must be there.
	if got := string(settings["after"]); got != "2" {
		t.Fatalf("after: %s", got)
	}
	// The delete removes the key it names.
	if _, ok := settings["before"]; ok {
		t.Fatal("the deleted key is still there")
	}
	if got := string(store["playlists"]["p:1"]); got != `{"id":"p1"}` {
		t.Fatalf("p:1: %s", got)
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

// TestDamagedStoreIsRefused says what is wrong with a store that does not
// hold together, instead of crashing on it: the import runs at startup, and a
// panic there is a crash loop.
func TestDamagedStoreIsRefused(t *testing.T) {
	t.Run("a log that ends one byte after its header", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "00001.mem")
		if err := os.WriteFile(path, make([]byte, vlogHeaderSize+1), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readWAL(path); err != nil {
			t.Fatalf("a torn log was refused: %v", err)
		}
	})
	t.Run("a log whose key length does not fit", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "00001.mem")
		raw := make([]byte, vlogHeaderSize)
		raw = append(raw, 0, 0)
		var length [binary.MaxVarintLen64]byte
		n := binary.PutUvarint(length[:], 1<<60)
		raw = append(raw, length[:n]...)
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readWAL(path); err != nil {
			t.Fatalf("a log with a length that does not fit was refused: %v", err)
		}
	})
	t.Run("a manifest that ends mid-varint", func(t *testing.T) {
		if changes := parseChanges([]byte{0x80}); len(changes) != 0 {
			t.Fatalf("changes = %v", changes)
		}
	})
	t.Run("a table whose index length does not fit", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "000001.sst")
		raw := make([]byte, 16)
		binary.BigEndian.PutUint32(raw[len(raw)-4:], 1<<30)
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readTable(path, 0); err == nil {
			t.Fatal("a table with an index that does not fit was read")
		}
	})
}
