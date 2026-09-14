package spotify

import (
	"errors"
	"testing"
)

func TestSpotifyIDValid(t *testing.T) {
	uri := "spotify:track:4uLU6hMCjMI75M1A2tKUQC"
	id, err := spotifyID(uri, "track")
	if err != nil {
		t.Fatalf("spotifyID: %v", err)
	}
	if id.Uri() != uri {
		t.Fatalf("Uri = %q, want %q", id.Uri(), uri)
	}
}

func TestSpotifyIDTypeMismatch(t *testing.T) {
	_, err := spotifyID("spotify:album:4uLU6hMCjMI75M1A2tKUQC", "track")
	var tm *TypeMismatchError
	if !errors.As(err, &tm) {
		t.Fatalf("want TypeMismatchError, got %v", err)
	}
	if tm.Expected != "track" || tm.Got != "album" {
		t.Fatalf("mismatch fields = %+v", tm)
	}
	if tm.Error() == "" {
		t.Fatal("empty error string")
	}
}

func TestSpotifyIDInvalidURI(t *testing.T) {
	if _, err := spotifyID("not-a-uri", "track"); err == nil {
		t.Fatal("want error for invalid URI")
	}
}
