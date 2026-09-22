package spotify

import (
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

var musicBucket = kv.UseBucket("spotify-music")

// cachedPlaylist pairs a playlist with the Spotify revision it was fetched at,
// so a background refresh can tell whether the playlist changed.
type cachedPlaylist struct {
	Revision string         `json:"revision"`
	Playlist music.Playlist `json:"playlist"`
}

func loadCachedPlaylist(playlistID string) (cachedPlaylist, bool) {
	if err := kv.EnsureStarted(); err != nil {
		return cachedPlaylist{}, false
	}
	var cached cachedPlaylist
	if err := musicBucket.GetObject("playlist:v3:"+playlistID, &cached); err != nil {
		return cachedPlaylist{}, false
	}
	return cached, true
}

func storeCachedPlaylist(playlistID string, cached cachedPlaylist) {
	if err := kv.EnsureStarted(); err != nil {
		return
	}
	_ = musicBucket.SetObject("playlist:v3:"+playlistID, cached)
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
	_ = musicBucket.Delete("playlist:v2:" + playlistID)
	return musicBucket.Delete("playlist:v3:" + playlistID)
}

func cachedUserPlaylists(user string, fetch func() ([]music.PlaylistSummary, error)) ([]music.PlaylistSummary, error) {
	return kv.Cached(musicBucket, "playlists:v3:"+user, fetch)
}

func revalidateUserPlaylists(user string) error {
	return musicBucket.Delete("playlists:v3:" + user)
}

func cachedUserProfile(username string, fetch func() (music.UserProfile, error)) (music.UserProfile, error) {
	return kv.Cached(musicBucket, "user:"+strings.ToLower(username), fetch)
}

func cachedArtist(uri string, fetch func() (music.ArtistInfo, error)) (music.ArtistInfo, error) {
	return kv.Cached(musicBucket, "artist:"+uri, fetch)
}

func cachedArtistDiscography(uri string, fetch func() (music.ArtistDiscography, error)) (music.ArtistDiscography, error) {
	return kv.Cached(musicBucket, "artist-discography:"+uri, fetch)
}

func cachedAlbumTracks(uri string, fetch func() ([]music.Song, error)) ([]music.Song, error) {
	return kv.Cached(musicBucket, "album:"+uri, fetch)
}

func cachedTrack(uri string, fetch func() (music.Song, error)) (music.Song, error) {
	return kv.Cached(musicBucket, "track:"+uri, fetch)
}
