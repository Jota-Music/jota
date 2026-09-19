//go:build discordlive

package app

import (
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
)

func TestLiveAppPresence(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	kv.Start()
	defer kv.Close()

	a := New("test")
	if err := a.SetDiscordEnabled(true); err != nil {
		t.Fatalf("enable: %v", err)
	}
	defer a.SetDiscordEnabled(false)

	if !a.DiscordEnabled() {
		t.Fatal("DiscordEnabled() = false after enabling")
	}

	payload := `{"title":"App path","artist":"Jota test","album":"Test","artwork":"","playing":true,"duration":200,"position":5}`
	if err := a.UpdateDiscordPresence(payload); err != nil {
		t.Fatalf("presence: %v", err)
	}
	t.Log("app-layer presence set as 'Jota' for 20s - check Discord now")
	time.Sleep(20 * time.Second)
}
