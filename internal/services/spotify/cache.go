package spotify

import (
	"github.com/Jota-Music/jota/internal/cache"
	"github.com/Jota-Music/jota/internal/music"
)

// playlistName is where a playlist is cached. The id is normalized first: the
// same playlist pasted as a URI and opened from the library must share one
// entry, or a revalidation of one form leaves the other stale.
func playlistName(playlistID string) string {
	return "spotify/playlists/" + normalizeID(playlistID, "playlist")
}

// contextName is where a user context is cached: a list that looks like a
// playlist but has no revision to compare.
func contextName(playlistID string) string { return "spotify/contexts/" + playlistID }

// libraryName is where the playlists an account owns are cached. The user is
// part of the name: two accounts on one device are two libraries.
func libraryName(user string) string { return "spotify/library/" + user }

// cachedPlaylist pairs a playlist with the Spotify revision it was fetched at,
// so a background refresh can tell whether the playlist changed.
type cachedPlaylist struct {
	Revision string         `json:"revision"`
	Playlist music.Playlist `json:"playlist"`
}

func loadCachedPlaylist(playlistID string) (cachedPlaylist, bool) {
	cached, ok := cache.Get[cachedPlaylist](playlistName(playlistID))
	if !ok {
		return cachedPlaylist{}, false
	}
	return cached, true
}

func storeCachedPlaylist(playlistID string, cached cachedPlaylist) {
	_ = cache.Put(playlistName(playlistID), cached)
}

// needsRevalidate reports whether the cached playlist is stale against the
// current revision. An unknown current revision can't be compared, so the cache
// wins; an empty cached revision means the entry predates revision tracking.
func needsRevalidate(cachedRevision, currentRevision string) bool {
	if currentRevision == "" {
		return false
	}
	return cachedRevision != currentRevision
}

func revalidateFullPlaylist(playlistID string) error {
	_ = cache.Forget(contextName(playlistID))
	return cache.Forget(playlistName(playlistID))
}

func revalidateUserPlaylists(user string) error {
	return cache.Forget(libraryName(user))
}
