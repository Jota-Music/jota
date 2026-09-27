package app

import (
	"testing"

	"github.com/Jota-Music/jota/internal/kv"
)

func TestRelayPersistence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	kv.Close()
	if err := kv.EnsureStarted(); err != nil {
		t.Fatalf("kv: %v", err)
	}
	t.Cleanup(kv.Close)

	app := &App{}
	if got := app.RelayConfig(); got.URL != "" {
		t.Fatalf("empty store reported %+v", got)
	}
	if err := app.SaveRelay("  ", "token"); err != nil {
		t.Fatalf("blank relay: %v", err)
	}
	if got := app.RelayConfig(); got.URL != "" {
		t.Fatalf("blank relay was stored: %+v", got)
	}
	if err := app.SaveRelay(" relay.example.com ", " tok "); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := app.RelayConfig(); got.URL != "relay.example.com" || got.Token != "tok" {
		t.Fatalf("stored = %+v", got)
	}

	pinned := &App{relay: Relay{URL: "dev.local"}}
	if err := pinned.SaveRelay("other.example.com", ""); err != nil {
		t.Fatalf("pinned save: %v", err)
	}
	if got := pinned.RelayConfig(); got.URL != "relay.example.com" {
		t.Fatalf("pin overwrote the saved relay: %+v", got)
	}
}
