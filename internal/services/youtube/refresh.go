package youtube

import (
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

// RefreshCatalog refetches every cached catalog entry in the background and
// rewrites only the ones that changed. Channel-id mappings are stable, so they
// are skipped; song entries outside the song:v2 prefix are audio source
// mappings, not catalog content.
func (s *Service) RefreshCatalog() error {
	if err := playlistBucket.ForEach(func(key string, _ []byte) error {
		switch {
		case strings.HasPrefix(key, "channel:playlists:"):
			id := strings.TrimPrefix(key, "channel:playlists:")
			kv.Refreshable(playlistBucket, key, func() ([]music.PlaylistSummary, error) {
				return fetchChannelTab(id, channelPlaylistsParams, func(section itemSection, _ string) []music.PlaylistSummary {
					return section.summaries()
				})
			})
		case strings.HasPrefix(key, "channel:videos:"):
			id := strings.TrimPrefix(key, "channel:videos:")
			kv.Refreshable(playlistBucket, key, func() ([]music.Song, error) {
				return fetchChannelTab(id, channelVideosParams, func(section itemSection, name string) []music.Song {
					return section.videos(id, name)
				})
			})
		case strings.HasPrefix(key, "channel:info:"):
			id := strings.TrimPrefix(key, "channel:info:")
			kv.Refreshable(playlistBucket, key, func() (music.ChannelInfo, error) {
				return channelInfo(id)
			})
		case strings.HasPrefix(key, "playlist:v3:"):
			id := strings.TrimPrefix(key, "playlist:v3:")
			kv.Refreshable(playlistBucket, key, func() (music.Playlist, error) {
				return fetchFullPlaylist(id)
			})
		}
		return nil
	}); err != nil {
		return err
	}

	return youtubeSourceBucket.ForEach(func(key string, _ []byte) error {
		if !strings.HasPrefix(key, "song:v2:") {
			return nil
		}
		id := strings.TrimPrefix(key, "song:v2:")
		kv.Refreshable(youtubeSourceBucket, key, func() (music.Song, error) {
			return fetchSong(id)
		})
		return nil
	})
}
