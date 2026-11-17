package music

import "testing"

func TestFollowIDsFollowTheNamingRule(t *testing.T) {
	if got, want := ArtistFollow(" 4uLU6hMCjMI75M1A2tKUQC "), "spotify:artist:4uLU6hMCjMI75M1A2tKUQC"; got != want {
		t.Fatalf("ArtistFollow = %q, want %q", got, want)
	}
	if got, want := ChannelFollow("youtube:UCY4YovmJwl4_BivQ-LkzU2A"), "youtube:channel:UCY4YovmJwl4_BivQ-LkzU2A"; got != want {
		t.Fatalf("ChannelFollow = %q, want %q", got, want)
	}
	if got, want := UserFollow("@aokihiro"), "spotify:user:aokihiro"; got != want {
		t.Fatalf("UserFollow = %q, want %q", got, want)
	}
}

func TestStoredFollowIDsAreCompleted(t *testing.T) {
	cases := map[string]string{
		"artist:4uLU6hMCjMI75M1A2tKUQC":                   "spotify:artist:4uLU6hMCjMI75M1A2tKUQC",
		"spotify:artist:4uLU6hMCjMI75M1A2tKUQC":           "spotify:artist:4uLU6hMCjMI75M1A2tKUQC",
		"spotify:artist:e5776bf7294146eba3c5a112dd5b6cd2": "spotify:artist:6YZOLlgZ38qVDSxVv9xg0G",
		"youtube:UCY4YovmJwl4_BivQ-LkzU2A":                "youtube:channel:UCY4YovmJwl4_BivQ-LkzU2A",
		"youtube:channel:UCY4YovmJwl4_BivQ-LkzU2A":        "youtube:channel:UCY4YovmJwl4_BivQ-LkzU2A",
		"user:aokihiro":         "spotify:user:aokihiro",
		"spotify:user:aokihiro": "spotify:user:aokihiro",
		"aokihiro":              "spotify:user:aokihiro",
		"local:playlist:abc":    "local:playlist:abc",
	}
	for id, want := range cases {
		if got := StoredFollowID(id); got != want {
			t.Fatalf("StoredFollowID(%q) = %q, want %q", id, got, want)
		}
	}
}
