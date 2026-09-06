package spotify

import (
	"jota/server/internal/kv"
	"jota/server/internal/music"
	"time"
)

var musicBucket = kv.UseBucket("spotify-music")

const musicCacheTTL = 12 * time.Hour

func cachedFullPlaylist(playlistID string, fetch func() (music.Playlist, error)) (music.Playlist, error) {
	if err := kv.EnsureStarted(); err != nil {
		return fetch()
	}

	var cached music.Playlist
	if err := musicBucket.GetObject("playlist:"+playlistID, &cached); err == nil {
		return cached, nil
	}

	playlist, err := fetch()
	if err != nil {
		return music.Playlist{}, err
	}

	_ = musicBucket.SetObject("playlist:"+playlistID, &playlist, musicCacheTTL)
	return playlist, nil
}

func revalidateFullPlaylist(playlistID string) error {
	return musicBucket.Delete("playlist:" + playlistID)
}

func cachedUserPlaylists(user string, fetch func() ([]music.PlaylistSummary, error)) ([]music.PlaylistSummary, error) {
	if err := kv.EnsureStarted(); err != nil {
		return fetch()
	}

	var cached []music.PlaylistSummary
	if err := musicBucket.GetObject("playlists:"+user, &cached); err == nil {
		return cached, nil
	}

	playlists, err := fetch()
	if err != nil {
		return nil, err
	}

	_ = musicBucket.SetObject("playlists:"+user, &playlists, musicCacheTTL)
	return playlists, nil
}

func revalidateUserPlaylists(user string) error {
	return musicBucket.Delete("playlists:" + user)
}
