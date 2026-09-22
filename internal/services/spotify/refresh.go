package spotify

import (
	"context"
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

// RefreshCatalog refetches every cached catalog entry in the background and
// rewrites only the ones that changed. Playlists keep their cheap revision
// check; everything else compares a fingerprint of the fetched content. Without
// a session the pass is skipped wholesale.
func (s *SpotifyService) RefreshCatalog() error {
	if s.Session() == nil {
		return nil
	}
	return musicBucket.ForEach(func(key string, _ []byte) error {
		switch {
		case strings.HasPrefix(key, "playlist:v2:"):
			s.refreshPlainPlaylist(strings.TrimPrefix(key, "playlist:v2:"))
		case strings.HasPrefix(key, "playlist:v3:"):
			s.refreshPlaylist(strings.TrimPrefix(key, "playlist:v3:"))
		case strings.HasPrefix(key, "playlists:v3:"):
			user := strings.TrimPrefix(key, "playlists:v3:")
			kv.Refreshable(musicBucket, key, func() ([]music.PlaylistSummary, error) {
				return s.userPlaylists(user)
			})
		case strings.HasPrefix(key, "user:"):
			username := strings.TrimPrefix(key, "user:")
			kv.Refreshable(musicBucket, key, func() (music.UserProfile, error) {
				return s.userProfile(username)
			})
		case strings.HasPrefix(key, "artist-discography:"):
			uri := strings.TrimPrefix(key, "artist-discography:")
			kv.Refreshable(musicBucket, key, func() (music.ArtistDiscography, error) {
				return s.artistDiscography(uri)
			})
		case strings.HasPrefix(key, "artist:"):
			uri := strings.TrimPrefix(key, "artist:")
			kv.Refreshable(musicBucket, key, func() (music.ArtistInfo, error) {
				return s.artist(uri)
			})
		case strings.HasPrefix(key, "album:"):
			uri := strings.TrimPrefix(key, "album:")
			kv.Refreshable(musicBucket, key, func() ([]music.Song, error) {
				return s.albumTracks(uri)
			})
		case strings.HasPrefix(key, "track:"):
			uri := strings.TrimPrefix(key, "track:")
			kv.Refreshable(musicBucket, key, func() (music.Song, error) {
				return s.song(uri)
			})
		}
		return nil
	})
}

func (s *SpotifyService) refreshPlainPlaylist(playlistID string) {
	kv.Refreshable(musicBucket, "playlist:v2:"+playlistID, func() (music.Playlist, error) {
		return s.fullPlaylist(playlistID)
	})
}

func (s *SpotifyService) refreshPlaylist(playlistID string) {
	cached, ok := loadCachedPlaylist(playlistID)
	if !ok {
		return
	}

	sess := s.Session()
	if sess == nil {
		return
	}

	uri := normalizeID(playlistID, "playlist")
	meta, err := getPlaylistMetadata(context.Background(), sess, uri)
	if err != nil || !needsRevalidate(cached.Revision, meta.revision) {
		return
	}

	playlist, err := s.playlistTracks(context.Background(), sess, uri, meta)
	if err != nil {
		return
	}
	storeCachedPlaylist(playlistID, cachedPlaylist{Revision: meta.revision, Playlist: playlist})
}
