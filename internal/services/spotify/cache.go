package spotify

import (
	"strings"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

var musicBucket = kv.UseBucket("spotify-music")

const musicCacheTTL = 6 * time.Hour

// cachedPlaylist pairs a playlist with the Spotify revision it was fetched at,
// so a later load can tell whether the playlist changed.
type cachedPlaylist struct {
	Revision string         `json:"revision"`
	Playlist music.Playlist `json:"playlist"`
}

func cachedFullPlaylist(playlistID string, fetch func() (music.Playlist, error)) (music.Playlist, error) {
	return kv.Cached(musicBucket, "playlist:v2:"+playlistID, musicCacheTTL, fetch)
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
	_ = musicBucket.SetObject("playlist:v3:"+playlistID, cached, musicCacheTTL)
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
	return kv.Cached(musicBucket, "playlists:v3:"+user, musicCacheTTL, fetch)
}

func revalidateUserPlaylists(user string) error {
	return musicBucket.Delete("playlists:v3:" + user)
}

func cachedUserProfile(username string, fetch func() (music.UserProfile, error)) (music.UserProfile, error) {
	return kv.Cached(musicBucket, "user:"+strings.ToLower(username), musicCacheTTL, fetch)
}

func cachedArtist(uri string, fetch func() (music.ArtistInfo, error)) (music.ArtistInfo, error) {
	return kv.Cached(musicBucket, "artist:"+uri, musicCacheTTL, fetch)
}

func cachedArtistDiscography(uri string, fetch func() (music.ArtistDiscography, error)) (music.ArtistDiscography, error) {
	return kv.Cached(musicBucket, "artist-discography:"+uri, musicCacheTTL, fetch)
}

func cachedAlbumTracks(uri string, fetch func() ([]music.Song, error)) ([]music.Song, error) {
	return kv.Cached(musicBucket, "album:"+uri, musicCacheTTL, fetch)
}

func cachedTrack(uri string, fetch func() (music.Song, error)) (music.Song, error) {
	return kv.Cached(musicBucket, "track:"+uri, musicCacheTTL, fetch)
}
