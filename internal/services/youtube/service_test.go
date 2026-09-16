package youtube

import (
	"testing"

	"github.com/Jota-Music/jota/internal/music"
)

func TestAudioQueryVariants(t *testing.T) {
	song := music.Song{
		Name:    "Regreso al sexo químicamente puro",
		Artists: []music.Artist{{Name: "Ilegales"}},
	}

	if got := audioQuery(song); got != "Regreso al sexo químicamente puro Ilegales" {
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
}
