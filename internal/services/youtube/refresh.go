package youtube

import (
	"strings"

	"github.com/Jota-Music/jota/internal/cache"
	"github.com/Jota-Music/jota/internal/music"
)

// RefreshCatalog refetches the cached catalog entries that can change in the
// background and rewrites only the ones that changed, answering whether anything
// did. Channel-id mappings are stable, so they are skipped; song entries outside
// the song:v2 prefix are audio source mappings, not catalog content. The signed-in
// account's playlist list is refreshed too, so a library that gained a playlist
// elsewhere catches up. Per-video metadata is left alone: it is the bulk of the
// cache (one entry per song) and revalidating it would be the whole library over
// the wire for a title that does not change.
func (s *Service) RefreshCatalog() (bool, error) {
	changed := false
	for _, dir := range []string{"youtube/library", "youtube/channels", "youtube/playlists", "youtube/videos"} {
		err := cache.Walk(dir, func(name string) error {
			changed = s.refreshEntity(name) || changed
			return nil
		})
		if err != nil {
			return changed, err
		}
	}
	return changed, nil
}

// refreshEntity revalidates one cached entity, rewriting it only when the
// source answered something different.
func (s *Service) refreshEntity(name string) bool {
	switch {
	case name == myPlaylistsKey:
		// A session the sign-in window handed over cannot sign a browse, so
		// refreshing it would only earn a 401.
		if sessionValue(cookies()) == "" {
			return false
		}
		return cache.Refreshable(name, fetchMyPlaylists)
	case strings.HasPrefix(name, "youtube/playlists/"):
		id := strings.TrimPrefix(name, "youtube/playlists/")
		return cache.Refreshable(name, func() ([]music.PlaylistSummary, error) {
			return fetchChannelTab(id, channelPlaylistsParams, func(section itemSection, _ string) []music.PlaylistSummary {
				return section.summaries()
			})
		})
	case strings.HasPrefix(name, "youtube/videos/"):
		id := strings.TrimPrefix(name, "youtube/videos/")
		return cache.Refreshable(name, func() ([]music.Song, error) {
			return fetchChannelTab(id, channelVideosParams, func(section itemSection, name string) []music.Song {
				return section.videos(id, name)
			})
		})
	case strings.HasPrefix(name, "youtube/channels/"):
		id := strings.TrimPrefix(name, "youtube/channels/")
		return cache.Refreshable(name, func() (music.ChannelInfo, error) {
			return channelInfo(id)
		})
	case strings.HasPrefix(name, "youtube/playlists/"):
		id := strings.TrimPrefix(name, "youtube/playlists/")
		return cache.Refreshable(name, func() (music.Playlist, error) {
			return fetchFullPlaylist(id)
		})
	}
	return false
}
