package spotify

import (
	"strings"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

var musicBucket = kv.UseBucket("spotify-music")

const musicCacheTTL = 12 * time.Hour

func cachedFullPlaylist(playlistID string, fetch func() (music.Playlist, error)) (music.Playlist, error) {
	return kv.Cached(musicBucket, "playlist:v2:"+playlistID, musicCacheTTL, fetch)
}

func revalidateFullPlaylist(playlistID string) error {
	return musicBucket.Delete("playlist:v2:" + playlistID)
}

func cachedUserPlaylists(user string, fetch func() ([]music.PlaylistSummary, error)) ([]music.PlaylistSummary, error) {
	return kv.Cached(musicBucket, "playlists:v2:"+user, musicCacheTTL, fetch)
}

func revalidateUserPlaylists(user string) error {
	return musicBucket.Delete("playlists:v2:" + user)
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
