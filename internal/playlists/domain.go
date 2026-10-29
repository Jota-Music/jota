package playlists

import "github.com/Jota-Music/jota/internal/music"

const defaultName = "Playlist"

// coverCount caps the mosaic a local playlist summary exposes.
const coverCount = 4

// coverResolveCap bounds the references a listing resolves when the playlist
// carries no metadata for them: the shelf waits on this call, so a source that
// answers nothing must not walk a whole playlist over the network.
const coverResolveCap = 16

// Playlist is a user-created playlist. It only stores references to the source
// tracks (a bare Spotify id or "youtube:<videoId>") plus the metadata known for
// them; a reference without stored metadata is resolved from its source.
type Playlist struct {
	Id         string                `json:"id"`
	Name       string                `json:"name"`
	Songs      []string              `json:"songs"`
	Tracks     map[string]music.Song `json:"tracks,omitempty"`
	ModifiedAt int64                 `json:"modifiedAt"`
}
