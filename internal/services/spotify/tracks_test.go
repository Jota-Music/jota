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

func TestGroupByURISharesDuplicates(t *testing.T) {
	const (
		trackA = "spotify:track:4uLU6hMCjMI75M1A2tKUQC"
		trackB = "spotify:track:3n3Ppam7vgaVa1iaRUc9Lp"
	)
	tracks := []Track{
		{URI: trackA},
		{URI: trackB},
		{URI: trackA},
		{URI: "not-a-uri"},
	}

	uris, index := groupByURI(tracks)

	if len(uris) != 2 {
		t.Fatalf("uris = %v, want 2 unique", uris)
	}
	if got := index[trackA]; len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("index[trackA] = %v, want [0 2]", got)
	}
	if _, ok := index["not-a-uri"]; ok {
		t.Fatal("invalid URI must be skipped")
	}
}
