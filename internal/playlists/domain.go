package playlists

import "github.com/Jota-Music/jota/internal/music"

const defaultName = "Playlist"

// coverCount caps the mosaic a local playlist summary exposes.
const coverCount = 4

// Playlist is a user-created playlist. It only stores references to the source
// tracks (a bare Spotify id or "youtube:<videoId>"); the metadata is resolved
// from the owning source when the playlist is opened.
type Playlist struct {
	Id    string   `json:"id"`
	Name  string   `json:"name"`
	Songs []string `json:"songs"`
}

// Resolver resolves a stored track reference to its full metadata. *music.Catalog
// satisfies it.
type Resolver interface {
	GetSong(id string) (music.Song, error)
}
