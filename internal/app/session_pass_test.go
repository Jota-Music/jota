package app

import (
	"encoding/json"
	"testing"

	"github.com/Jota-Music/jota/internal/backup"
	"github.com/Jota-Music/jota/internal/services/spotify"
	"github.com/Jota-Music/jota/internal/services/youtube"
)

// A session that reached the document before this device knew the pass has to be
// opened as soon as the pass is set: the pass is the only thing it was waiting
// for, and the panel says so in the meantime instead of staying silent.
func TestSessionPassAppliesStagedBlob(t *testing.T) {
	a := syncApp(t)
	// Setting the pass queues a background pass, and a pass that outlives the test
	// writes into a store whose directory is already gone. The test is about what
	// the call applies, not about that pass.
	if err := a.SetSyncAuto(false); err != nil {
		t.Fatalf("disable autosave: %v", err)
	}
	t.Setenv("YOUTUBE_COOKIES", "")
	_ = youtube.ClearCookies()
	t.Cleanup(func() { _ = youtube.ClearCookies() })

	const cookies = "SAPISID=staged; __Secure-3PSID=full"
	blob, err := encryptWithPass("shared", cookies)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	value, err := json.Marshal(blob)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := sessionsSync.Apply(backup.Settings{"youtube": {Value: value, At: 1}}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if got := youtube.Cookies(); got != "" {
		t.Fatalf("a session was applied without the pass: %q", got)
	}
	if !a.SyncStatus().Sealed {
		t.Fatal("a session the device cannot open was not reported as sealed")
	}

	if err := a.SetSessionPass("shared"); err != nil {
		t.Fatalf("set pass: %v", err)
	}
	if got := youtube.Cookies(); got != cookies {
		t.Fatalf("staged session = %q, want %q", got, cookies)
	}
	if a.SyncStatus().Sealed {
		t.Fatal("a pass that opened the staged session still reports it as sealed")
	}
}

// Staging a session this device already staged is not news: a fresh salt would
// change the blob, mark the pass dirty and make the other devices re-open a
// session that never moved.
func TestSyncSessionsAreNotStagedWhenUnchanged(t *testing.T) {
	a := syncApp(t)
	// Setting the pass and every later stage ask for a background pass, and a pass
	// that outlives the test writes into a store whose directory is already gone.
	if err := a.SetSyncAuto(false); err != nil {
		t.Fatalf("disable autosave: %v", err)
	}
	if err := a.SetSessionPass("shared"); err != nil {
		t.Fatalf("set pass: %v", err)
	}
	t.Setenv("YOUTUBE_COOKIES", "")
	_ = youtube.ClearCookies()
	t.Cleanup(func() { _ = youtube.ClearCookies() })

	staged := func(key string) (string, int64) {
		t.Helper()
		entry, ok := stagedBlob(key)
		if !ok || entry.Value == "" {
			t.Fatalf("staged %s session missing", key)
		}
		if entry.At == 0 {
			t.Fatalf("staged %s session has no write time", key)
		}
		return entry.Value, entry.At
	}
	unchanged := func(key, blob string, at int64) {
		t.Helper()
		if sessionsSync.Pending() {
			t.Fatalf("restaging the same %s session marked the pass dirty", key)
		}
		got, moved := staged(key)
		if got != blob {
			t.Fatalf("restaging the same %s session re-encrypted the blob", key)
		}
		if moved != at {
			t.Fatalf("restaging the same %s session moved the write time from %d to %d", key, at, moved)
		}
	}

	const cookies = "SAPISID=staged; __Secure-3PSID=full"
	if err := youtube.SetCookies(cookies); err != nil {
		t.Fatalf("set cookies: %v", err)
	}
	a.syncYouTubeCookies()
	blob, at := staged("youtube")
	sessionsSync.Synced()
	a.syncYouTubeCookies()
	unchanged("youtube", blob, at)

	if err := spotify.StoreCredentials("alice", []byte("abcdef")); err != nil {
		t.Fatalf("store spotify session: %v", err)
	}
	a.syncSpotifySession()
	blob, at = staged("spotify")
	sessionsSync.Synced()
	a.syncSpotifySession()
	unchanged("spotify", blob, at)
}
