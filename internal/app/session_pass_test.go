package app

import (
	"encoding/json"
	"testing"

	"github.com/Jota-Music/jota/internal/backup"
	"github.com/Jota-Music/jota/internal/services/youtube"
)

// A session that reached the document before this device knew the pass has to be
// opened as soon as the pass is set: the pass is the only thing it was waiting
// for, and the panel says so in the meantime instead of staying silent.
func TestSessionPassAppliesStagedBlob(t *testing.T) {
	a := syncApp(t)
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
