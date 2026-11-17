package music

import "testing"

func TestRoomIDsFollowTheNamingRule(t *testing.T) {
	if got, want := RoomID(" Relay.Example.com ", " party "), "local:room:relay.example.com|party"; got != want {
		t.Fatalf("RoomID = %q, want %q", got, want)
	}

	composed := "local:room:relay.example.com|party"
	if got := StoredRoomID(composed); got != composed {
		t.Fatalf("StoredRoomID(%q) = %q", composed, got)
	}
	if got := StoredRoomID("relay.example.com|party"); got != composed {
		t.Fatalf("legacy id = %q, want %q", got, composed)
	}
	// An id that names no relay stays as it is, so a page-only key is not
	// rewritten into a room.
	if got := StoredRoomID("active-room"); got != "active-room" {
		t.Fatalf("bare key = %q", got)
	}
}
