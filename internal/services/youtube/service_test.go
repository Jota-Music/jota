package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/music"
)

func TestAudioQueryVariants(t *testing.T) {
	// The album rides along on the album-first query and is dropped from the
	// artist-first one, which stays broad enough to recover.
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

	// Both queries run together, so the album-first one cannot bury the song.
	if got := queries(song); len(got) != 2 || got[0] == got[1] {
		t.Fatalf("queries = %q, want the distinct album-first and artist-first searches", got)
	}
	if got := queries(music.Song{Name: "Solo"}); len(got) != 1 {
		t.Fatalf("queries = %q, want a single search when there is no album", got)
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
	audio, err := svc.searchAudio(music.Song{Id: songID, Name: "Lo que quieras"})
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
	if _, err := svc.searchAudio(music.Song{Id: "test-no-link", Name: "Solo"}); err == nil {
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
	if _, err := svc.searchAudio(music.Song{Id: songID, Name: "Solo"}); err == nil {
		t.Fatal("expected an error when the linked id is unplayable and the search is empty")
	}
	if *searches != 1 {
		t.Fatalf("searched %d times, want the fallback search", *searches)
	}
}

// searchVideos builds the innertube envelope extractVideos decodes, so a test
// can push candidates through the real search and ranking path.
func searchVideos(videos ...Video) searchResponse {
	items := make([]string, 0, len(videos))
	for _, v := range videos {
		badge := ""
		if v.Official {
			badge = `,"ownerBadges":[{"metadataBadgeRenderer":{"icon":{"iconType":"OFFICIAL_ARTIST_BADGE"},"style":"BADGE_STYLE_TYPE_VERIFIED_ARTIST"}}]`
		}
		items = append(items, fmt.Sprintf(
			`{"compactVideoRenderer":{"videoId":%q,"title":{"runs":[{"text":%q}]},"shortBylineText":{"runs":[{"text":%q}]},"lengthText":{"runs":[{"text":%q}]}%s}}`,
			v.ID, v.Title, v.Author, fmt.Sprintf("%d:%02d", v.Duration/60, v.Duration%60), badge))
	}
	var sr searchResponse
	_ = json.Unmarshal([]byte(`{"contents":{"sectionListRenderer":{"contents":[{"itemSectionRenderer":{"contents":[`+strings.Join(items, ",")+`]}}]}}}`), &sr)
	return sr
}

// stubSearch swaps the innertube search for canned responses keyed by query.
func stubSearch(t *testing.T, responses map[string]searchResponse) {
	t.Helper()
	old := searchFn
	searchFn = func(_, query, _ string) (searchResponse, error) {
		return responses[query], nil
	}
	t.Cleanup(func() { searchFn = old })
}

// An album sharing its name with another track makes the album-first query bury
// the song; merging the artist-first query is what surfaces the official upload.
func TestCandidatesMergesAlbumAndArtistQueries(t *testing.T) {
	const songID = "test-merge-queries"
	_ = youtubeSourceBucket.Delete(songID)

	song := music.Song{
		Id:       songID,
		Name:     "Rambo",
		Artists:  []music.Artist{{Name: "Orquesta Mondragón"}},
		Album:    music.Album{Title: "Ellos las prefieren gordas"},
		Duration: 182,
	}

	stubSearch(t, map[string]searchResponse{
		audioQuery(song): searchVideos(
			Video{ID: "w2qb3mifRro", Title: "Orquesta Mondragón - Rambo (Sábado noche'10.10.87) (Color Remastered)", Author: "romelena romelena", Duration: 179},
			Video{ID: "gordasTrack", Title: "Ellos las prefieren gordas - Orquesta Mondragon", Author: "Hank Mc Koy", Duration: 221},
		),
		audioQueryArtistFirst(song): searchVideos(
			Video{ID: "ohV8ZwFXQN4", Title: "Rambo", Author: "Orquesta Mondragón - Topic", Duration: 183},
		),
	})

	videos, err := NewService().candidates(songID, song)
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	if len(videos) == 0 || videos[0].ID != "ohV8ZwFXQN4" {
		t.Fatalf("first = %+v, want the official from the artist-first query", videos)
	}
}

// One query failing must not sink a resolve the other one can satisfy.
func TestCandidatesKeepsResultsWhenOneQueryFails(t *testing.T) {
	const songID = "test-merge-partial"
	_ = youtubeSourceBucket.Delete(songID)

	song := music.Song{
		Id:      songID,
		Name:    "Rambo",
		Artists: []music.Artist{{Name: "Orquesta Mondragón"}},
		Album:   music.Album{Title: "Otra"},
	}

	old := searchFn
	searchFn = func(_, query, _ string) (searchResponse, error) {
		if query == audioQuery(song) {
			return searchResponse{}, errors.New("boom")
		}
		return searchVideos(Video{ID: "ohV8ZwFXQN4", Title: "Rambo"}), nil
	}
	t.Cleanup(func() { searchFn = old })

	videos, err := NewService().candidates(songID, song)
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	if len(videos) == 0 || videos[0].ID != "ohV8ZwFXQN4" {
		t.Fatalf("first = %+v, want the surviving query's result", videos)
	}
}
