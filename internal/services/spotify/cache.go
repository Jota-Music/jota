package spotify

import (
	"jota/server/internal/kv"
	"jota/server/internal/music"
	"time"
)

var musicBucket = kv.UseBucket("spotify-music")

const musicCacheTTL = 12 * time.Hour

func cachedFullPlaylist(playlistID string, fetch func() (music.Playlist, error)) (music.Playlist, error) {
	return kv.Cached(musicBucket, "playlist:"+playlistID, musicCacheTTL, fetch)
}

func revalidateFullPlaylist(playlistID string) error {
	return musicBucket.Delete("playlist:" + playlistID)
}

func cachedUserPlaylists(user string, fetch func() ([]music.PlaylistSummary, error)) ([]music.PlaylistSummary, error) {
	return kv.Cached(musicBucket, "playlists:"+user, musicCacheTTL, fetch)
}

func revalidateUserPlaylists(user string) error {
	return musicBucket.Delete("playlists:" + user)
}
