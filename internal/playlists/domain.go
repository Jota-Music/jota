package playlists

import "github.com/Jota-Music/jota/internal/music"

const defaultName = "Playlist"

// coverCount caps the mosaic a local playlist summary exposes.
const coverCount = 4

// Playlist is a user-created playlist. It only stores references to the source
// tracks (a bare Spotify id or "youtube:<videoId>"); the metadata is resolved
// from the owning source when the playlist is opened.
type Playlist struct {
	Id         string                `json:"id"`
	Name       string                `json:"name"`
	Songs      []string              `json:"songs"`
	Tracks     map[string]music.Song `json:"tracks,omitempty"`
	ModifiedAt int64                 `json:"modifiedAt"`
}
