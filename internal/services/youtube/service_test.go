package youtube

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/music"
)

func TestAudioQueryVariants(t *testing.T) {
	// The album rides along on the primary query but not on the artist-first
	// fallback, which has to stay broad enough to recover.
	song := music.Song{
		Name:    "Regreso al sexo químicamente puro",
		Artists: []music.Artist{{Name: "Ilegales"}},
		Album:   music.Album{Title: "Regreso al sexo químicamente puro"},
	}

	if got := audioQuery(song); got != "Regreso al sexo químicamente puro Ilegales Regreso al sexo químicamente puro" {
		t.Fatalf("audioQuery = %q", got)
	}
	if got := audioQueryArtistFirst(song); got != "Ilegales Regreso al sexo químicamente puro" {
		t.Fatalf("audioQueryArtistFirst = %q", got)
	}

	noArtist := music.Song{Name: "Solo"}
	if got := audioQueryArtistFirst(noArtist); got != "Solo" {
		t.Fatalf("no artists = %q, want Solo", got)
	}
	if got := audioQuery(noArtist); got != "Solo" {
		t.Fatalf("no artists = %q, want Solo", got)
	}

	// A song with neither artist nor album is just its title, and a blank album
	// must not leave a trailing space.
	noAlbum := music.Song{Name: "Lo que quieras", Artists: []music.Artist{{Name: "Jere"}}, Album: music.Album{Title: "  "}}
	if got := audioQuery(noAlbum); got != "Lo que quieras Jere" {
		t.Fatalf("blank album = %q", got)
	}
}

// countingSearch swaps the innertube search for a counter so a test can tell
// whether a resolve reached the network.
func countingSearch(t *testing.T, result searchResponse) *int {
	t.Helper()
	old := searchFn
	searches := 0
	searchFn = func(_, _, _ string) (searchResponse, error) {
		searches++
		return result, nil
	}
	t.Cleanup(func() { searchFn = old })
	return &searches
}

func playableServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestSearchAudioPlaysTheLinkedIdWithoutSearching(t *testing.T) {
	const songID = "test-linked-audio"
	const videoID = "linkedVideo1"

	_ = youtubeSourceBucket.SetString(songID, videoID)
	_ = audioBucket.SetObject(videoID, music.Audio{
		Url:      playableServer(t),
		VideoID:  videoID,
		ExpireAt: time.Now().Add(time.Hour).Unix(),
	}, time.Hour)

	searches := countingSearch(t, searchResponse{})

	svc := NewService()
	audio, err := svc.searchAudio(music.Song{Id: songID, Name: "Lo que quieras"}, "Lo que quieras Jere Guernica")
	if err != nil {
		t.Fatalf("searchAudio: %v", err)
	}
	if audio.VideoID != videoID {
		t.Fatalf("video = %q, want the linked %q", audio.VideoID, videoID)
	}
	if *searches != 0 {
		t.Fatalf("searched %d times, want 0 for a linked id", *searches)
	}
}

func TestSearchAudioSearchesWithoutALinkedId(t *testing.T) {
	searches := countingSearch(t, searchResponse{})

	svc := NewService()
	if _, err := svc.searchAudio(music.Song{Id: "test-no-link", Name: "Solo"}, "Solo"); err == nil {
		t.Fatal("expected an error when the search finds nothing")
	}
	if *searches != 1 {
		t.Fatalf("searched %d times, want 1", *searches)
	}
}

func TestSearchAudioFallsBackToSearchWhenTheLinkedIdDies(t *testing.T) {
	const songID = "test-dead-link"
	const videoID = "deadVideo01"

	_ = youtubeSourceBucket.SetString(songID, videoID)

	oldPlayer := playerStreamURLFn
	playerStreamURLFn = func(clientConfig, string) (string, error) {
		return "", errors.New("nope")
	}
	t.Cleanup(func() { playerStreamURLFn = oldPlayer })

	searches := countingSearch(t, searchResponse{})

	svc := NewService()
	if _, err := svc.searchAudio(music.Song{Id: songID, Name: "Solo"}, "Solo"); err == nil {
		t.Fatal("expected an error when the linked id is unplayable and the search is empty")
	}
	if *searches != 1 {
		t.Fatalf("searched %d times, want the fallback search", *searches)
	}
}
