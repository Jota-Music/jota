package spotify

import (
	"fmt"

	librespot "github.com/devgianlu/go-librespot"
)

type Playlist struct {
	URI        string
	Name       string
	Owner      string
	CoverURL   string
	TrackCount int32 // -1 if not available
}

type Track struct {
	URI        string
	Name       string
	Artists    []string
	ArtistURIs []string
	Album      string
	AlbumURI   string
	CoverURL   string
	Duration   int
}

type AlbumRef struct {
	URI      string
	Name     string
	Year     int32
	CoverURL string
	Group    string
}

type ArtistInfo struct {
	Name     string
	URI      string
	ImageURL string
	Tracks   []Track
}

type ArtistDiscography struct {
	Name   string
	URI    string
	Albums []AlbumRef
}

type TypeMismatchError struct {
	Expected string
	Got      librespot.SpotifyIdType
	URI      string
}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("expected spotify:%s:..., got spotify:%s: (%s)", e.Expected, e.Got, e.URI)
}
