package spotify

import "testing"

// A degraded service must keep reporting the account it caches under. The
// frontend keys its queries by Username() and falls back to "default" on an
// empty string, which would silently read a cache bucket that does not exist.
func TestUsernameSurvivesDegraded(t *testing.T) {
	s := NewSpotifyService("")
	if got := s.Username(); got != "" {
		t.Fatalf("fresh service username = %q, want empty", got)
	}

	s.username = "listener"
	s.degraded = true

	if got := s.Username(); got != "listener" {
		t.Errorf("degraded username = %q, want %q", got, "listener")
	}
	if s.IsConnected() {
		t.Error("degraded service reports connected; writes need a live session")
	}
	if !s.IsDegraded() {
		t.Error("IsDegraded() = false, want true")
	}
}

func TestDegradedRequiresStoredAccount(t *testing.T) {
	s := NewSpotifyService("")
	if s.IsDegraded() {
		t.Error("a service with no stored credentials must not claim to be degraded")
	}
	if s.isFinished() {
		t.Error("fresh service reports finished")
	}
}
