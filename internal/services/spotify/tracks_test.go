package spotify

import (
	"context"
	"slices"
	"testing"

	librespot "github.com/devgianlu/go-librespot"
	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"

	"github.com/Jota-Music/jota/internal/music"
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

func TestContextTracksFromSearch(t *testing.T) {
	resolved := &connectpb.Context{
		Uri: "spotify:search:Daft+Punk",
		Pages: []*connectpb.ContextPage{{
			Tracks: []*connectpb.ContextTrack{{Gid: make([]byte, 16)}},
		}},
	}

	got, err := contextTracks(t.Context(), nil, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].GetUri() != "spotify:track:0000000000000000000000" {
		t.Fatalf("contextTracks = %+v", got)
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

// Song ids carry the spotify: prefix so every source can be routed by id, and
// every id-based call depends on that staying true, so pin the contract here.
func TestSongIdIsPrefixed(t *testing.T) {
	const uri = "spotify:track:4uLU6hMCjMI75M1A2tKUQC"
	song := trackToSong(Track{URI: uri, Name: "Test"})

	if song.Id != "spotify:4uLU6hMCjMI75M1A2tKUQC" {
		t.Fatalf("Song.Id = %q, want the prefixed id", song.Id)
	}
}

// Track.Duration is milliseconds; the frontend formats seconds, so the search
// payload must convert.
func TestSearchResultDurationIsSeconds(t *testing.T) {
	got := trackToSearchResult(Track{URI: "spotify:track:x", Duration: 238_000})

	if got.Duration != 238 {
		t.Fatalf("SearchResult.Duration = %d, want 238 seconds", got.Duration)
	}
}

// Artist ids must keep the spotify: prefix and stay paired by position, or the
// frontend cannot route to the artist page.
func TestSearchResultArtistsAreBare(t *testing.T) {
	got := trackToSearchResult(Track{
		URI:        "spotify:track:x",
		Artists:    []string{"Radiohead", "Radiohead"},
		ArtistURIs: []string{"spotify:artist:4Z8W4fKeB5YxbusRsdQVPb"},
	})

	want := []music.Artist{
		{Id: "spotify:4Z8W4fKeB5YxbusRsdQVPb", Name: "Radiohead"},
		{Name: "Radiohead"},
	}
	if !slices.Equal(got.Artists, want) {
		t.Fatalf("SearchResult.Artists = %+v, want %+v", got.Artists, want)
	}
}

// The autoplay request is a station context built from the seed track, so every
// form the UI or a playlist id hands over must land on the same track uri.
func TestRadioRequestUsesTrackURI(t *testing.T) {
	const trackID = "4uLU6hMCjMI75M1A2tKUQC"
	for _, id := range []string{
		"spotify:" + trackID,
		URITrackPrefix + trackID,
		URIRadioPrefix + trackID,
		URIStationPrefix + trackID,
		URIRadioPrefix + "spotify:" + trackID,
		URIStationPrefix + "spotify:" + trackID,
	} {
		if got := seed(id); got != trackID {
			t.Fatalf("seed(%q) = %q, want %q", id, got, trackID)
		}
		if got := URIStationPrefix + seed(id); got != "spotify:station:track:"+trackID {
			t.Fatalf("station uri = %q", got)
		}
	}
}

// A radio context has no io.EOF: every page names a next one, so the walk must
// stop on the page bound instead of running forever.
func TestWalkStopsAtRadioMaxPages(t *testing.T) {
	src := &endlessStation{}

	got, err := walk(t.Context(), src, radioMaxPages)
	if err != nil {
		t.Fatal(err)
	}
	if src.read != radioMaxPages {
		t.Fatalf("pages read = %d, want radioMaxPages (%d)", src.read, radioMaxPages)
	}
	if len(got) != radioMaxPages {
		t.Fatalf("walk = %d tracks, want %d", len(got), radioMaxPages)
	}
}

type endlessStation struct{ read int }

func (e *endlessStation) Type() librespot.SpotifyIdType {
	return librespot.SpotifyIdTypeTrack
}

func (e *endlessStation) Page(context.Context, int) ([]*connectpb.ContextTrack, error) {
	e.read++
	return []*connectpb.ContextTrack{{Uri: "spotify:track:4uLU6hMCjMI75M1A2tKUQC"}}, nil
}
