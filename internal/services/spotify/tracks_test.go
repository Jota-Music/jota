package spotify

import (
	"testing"

	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"
)

func TestTracksFromContext(t *testing.T) {
	cts := []*connectpb.ContextTrack{
		{Uri: "spotify:track:a"},
		{Uri: "spotify:local:x"},
		{},
		{Uri: "spotify:track:b"},
	}

	got := tracksFromContext(cts)

	if len(got) != 2 || got[0].URI != "spotify:track:a" || got[1].URI != "spotify:track:b" {
		t.Fatalf("tracksFromContext = %+v", got)
	}
}
