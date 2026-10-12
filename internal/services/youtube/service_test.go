package youtube

import (
	"testing"

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
