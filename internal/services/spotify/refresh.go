package spotify

import (
	"context"
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

// RefreshCatalog refetches the cached catalog entries that can change in the
// background and rewrites only the ones that changed, answering whether anything
// did. Playlists keep their cheap revision check; everything else compares a
// fingerprint of the fetched content. Per-track metadata is left alone: it is the
// bulk of the cache (one entry per song) and a track's title, duration and cover
// do not change, so revalidating them would be the whole library over the wire
// for nothing. Without a live session the pass is skipped wholesale, which is what
// lets a degraded service keep serving a cache that only ages until Watch
// reconnects.
func (s *SpotifyService) RefreshCatalog() (bool, error) {
	if s.Session() == nil {
		return false, nil
	}
	changed := false
	err := musicBucket.ForEach(func(key string, _ []byte) error {
		switch {
		case strings.HasPrefix(key, "playlist:v2:"):
			changed = s.refreshPlainPlaylist(strings.TrimPrefix(key, "playlist:v2:")) || changed
		case strings.HasPrefix(key, "playlist:v3:"):
			changed = s.refreshPlaylist(strings.TrimPrefix(key, "playlist:v3:")) || changed
		case strings.HasPrefix(key, "playlists:v3:"):
			user := strings.TrimPrefix(key, "playlists:v3:")
			changed = kv.Refreshable(musicBucket, key, func() ([]music.PlaylistSummary, error) {
				return s.userPlaylists(user)
			}) || changed
		case strings.HasPrefix(key, "user:"):
			username := strings.TrimPrefix(key, "user:")
			changed = kv.Refreshable(musicBucket, key, func() (music.UserProfile, error) {
				return s.userProfile(username)
			}) || changed
		case strings.HasPrefix(key, "following:v1:"):
			changed = kv.Refreshable(musicBucket, key, s.follows) || changed
		case strings.HasPrefix(key, "artist-discography:"):
			uri := strings.TrimPrefix(key, "artist-discography:")
			changed = kv.Refreshable(musicBucket, key, func() (music.ArtistDiscography, error) {
				return s.artistDiscography(uri)
			}) || changed
		case strings.HasPrefix(key, "artist:"):
			uri := strings.TrimPrefix(key, "artist:")
			changed = kv.Refreshable(musicBucket, key, func() (music.ArtistInfo, error) {
				return s.artist(uri)
			}) || changed
		case strings.HasPrefix(key, "album:"):
			uri := strings.TrimPrefix(key, "album:")
			changed = kv.Refreshable(musicBucket, key, func() ([]music.Song, error) {
				return s.albumTracks(uri)
			}) || changed
		}
		return nil
	})
	return changed, err
}

func (s *SpotifyService) refreshPlainPlaylist(playlistID string) bool {
	return kv.Refreshable(musicBucket, "playlist:v2:"+playlistID, func() (music.Playlist, error) {
		return s.fullPlaylist(playlistID)
	})
}

func (s *SpotifyService) refreshPlaylist(playlistID string) bool {
	cached, ok := loadCachedPlaylist(playlistID)
	if !ok {
		return false
	}

	sess := s.Session()
	if sess == nil {
		return false
	}

	uri := normalizeID(playlistID, "playlist")
	meta, err := getPlaylistMetadata(context.Background(), sess, uri)
	if err != nil || !needsRevalidate(cached.Revision, meta.revision) {
		return false
	}

	playlist, err := s.playlistTracks(context.Background(), sess, uri, meta)
	if err != nil {
		return false
	}
	storeCachedPlaylist(playlistID, cachedPlaylist{Revision: meta.revision, Playlist: playlist})
	return true
}
