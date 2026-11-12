package spotify

import (
	"context"
	"strings"

	"github.com/Jota-Music/jota/internal/cache"
	"github.com/Jota-Music/jota/internal/music"
)

// refreshDirs are the directories the refresh walks. Each holds one kind of
// cached entity, so the file names say what they are.
var refreshDirs = []string{"spotify/library", "spotify/playlists", "spotify/contexts", "spotify/users", "spotify/follows", "spotify/friends", "spotify/discography", "spotify/artists", "spotify/albums"}

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
	for _, dir := range refreshDirs {
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
func (s *SpotifyService) refreshEntity(name string) bool {
	switch {
	case strings.HasPrefix(name, "spotify/playlists/"):
		return s.refreshPlaylist(strings.TrimPrefix(name, "spotify/playlists/"))
	case strings.HasPrefix(name, "spotify/contexts/"):
		return s.refreshContext(strings.TrimPrefix(name, "spotify/contexts/"))
	case strings.HasPrefix(name, "spotify/library/"):
		// The entry belongs to the account that wrote it.
		if strings.TrimPrefix(name, "spotify/library/") != s.Username() {
			return false
		}
		return cache.Refreshable(name, func() ([]music.PlaylistSummary, error) {
			return s.userPlaylists(s.Username())
		})
	case strings.HasPrefix(name, "spotify/users/"):
		username := strings.TrimPrefix(name, "spotify/users/")
		return cache.Refreshable(name, func() (music.UserProfile, error) {
			return s.userProfile(username)
		})
	case strings.HasPrefix(name, "spotify/follows/"):
		// The entry belongs to the account that wrote it: a session for
		// another account would answer with its own follows.
		if strings.TrimPrefix(name, "spotify/follows/") != s.Username() {
			return false
		}
		return cache.Refreshable(name, s.follows)
	case strings.HasPrefix(name, "spotify/friends/"):
		if strings.TrimPrefix(name, "spotify/friends/") != s.Username() {
			return false
		}
		return cache.Refreshable(name, s.friends)
	case strings.HasPrefix(name, "spotify/discography/"):
		uri := strings.TrimPrefix(name, "spotify/discography/")
		return cache.Refreshable(name, func() (music.ArtistDiscography, error) {
			return s.artistDiscography(uri)
		})
	case strings.HasPrefix(name, "spotify/artists/"):
		uri := strings.TrimPrefix(name, "spotify/artists/")
		return cache.Refreshable(name, func() (music.ArtistInfo, error) {
			return s.artist(uri)
		})
	case strings.HasPrefix(name, "spotify/albums/"):
		uri := strings.TrimPrefix(name, "spotify/albums/")
		return cache.Refreshable(name, func() ([]music.Song, error) {
			return s.albumTracks(uri)
		})
	}
	return false
}

func (s *SpotifyService) refreshContext(playlistID string) bool {
	return cache.Refreshable(contextName(playlistID), func() (music.Playlist, error) {
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
