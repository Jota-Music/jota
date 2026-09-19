//go:build discordlive

package discord

import (
	"os"
	"testing"
	"time"
)

func TestLiveActivity(t *testing.T) {
	clientID := os.Getenv("DISCORD_CLIENT_ID")
	if clientID == "" {
		clientID = "1550829418388131892"
	}
	s := New(clientID)
	defer s.Close()

	activity := Activity{
		Type:    2,
		Details: "Smoke test",
		State:   "discordlive",
	}
	if err := s.Set(activity); err != nil {
		t.Fatalf("connect/set failed (is Discord running?): %v", err)
	}
	t.Log("activity set as 'Jota' for 20s - open your Discord profile popout now")
	time.Sleep(20 * time.Second)

	if err := s.Clear(); err != nil {
		t.Fatalf("clear failed: %v", err)
	}
}
