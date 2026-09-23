package spotify

import (
	"context"
	"fmt"
	"strings"

	"github.com/devgianlu/go-librespot/session"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

func (s *SpotifyService) GetFullPlaylist(playlistID string) (music.Playlist, error) {
	// Radio is a synthetic context, not a playlist: never cached, recomputed on
	// every visit.
	if strings.HasPrefix(playlistID, URIRadioPrefix) || strings.HasPrefix(playlistID, URIStationPrefix) {
		return s.GetSongRadio(playlistID)
	}

	uri := normalizeID(playlistID, "playlist")

	// Liked Songs and other user contexts aren't playlists: no revision to
	// compare, so they get a plain eternal cache entry.
	if !strings.HasPrefix(uri, URIPlaylistPrefix) {
		return kv.Cached(musicBucket, "playlist:v2:"+playlistID, func() (music.Playlist, error) {
			return s.fullPlaylist(playlistID)
		})
	}

	// Serve what's stored; freshness is the refresh loop's job.
	if cached, hasCached := loadCachedPlaylist(playlistID); hasCached {
		return cached.Playlist, nil
	}

	sess := s.Session()
	if sess == nil {
		return music.Playlist{}, ErrNotConnected
	}

	meta, err := getPlaylistMetadata(context.Background(), sess, uri)
	if err != nil {
		return music.Playlist{}, err
	}
	playlist, err := s.playlistTracks(context.Background(), sess, uri, meta)
	if err != nil {
		return music.Playlist{}, err
	}

	storeCachedPlaylist(playlistID, cachedPlaylist{Revision: meta.revision, Playlist: playlist})
	return playlist, nil
}

func (s *SpotifyService) RevalidateFullPlaylist(playlistID string) error {
	return revalidateFullPlaylist(playlistID)
}

func (s *SpotifyService) fullPlaylist(playlistID string) (music.Playlist, error) {
	uri := normalizeID(playlistID, "playlist")

	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return music.Playlist{}, ErrNotConnected
	}

	meta, _ := getPlaylistMetadata(ctx, sess, uri)
	return s.playlistTracks(ctx, sess, uri, meta)
}

func (s *SpotifyService) playlistTracks(ctx context.Context, sess *session.Session, uri string, meta playlistMeta) (music.Playlist, error) {
	ctxTracks, err := resolveContextTracks(ctx, sess, uri)
	if err != nil {
		return music.Playlist{}, fmt.Errorf("resolve playlist: %w", err)
	}

	tracks := tracksFromContext(ctxTracks)
	enrichTracks(ctx, sess, tracks)

	allSongs := make([]music.Song, 0, len(tracks))
	for _, t := range tracks {
		allSongs = append(allSongs, trackToSong(t))
	}

	return music.Playlist{
		Name:  meta.name,
		Cover: meta.cover,
		Owner: meta.owner,
		Songs: allSongs,
	}, nil
}

func (s *SpotifyService) GetUserPlaylists(user string) ([]music.PlaylistSummary, error) {
	return kv.Cached(musicBucket, "playlists:v3:"+user, func() ([]music.PlaylistSummary, error) {
		return s.userPlaylists(user)
	})
}

func (s *SpotifyService) RevalidateUserPlaylists(user string) error {
	return revalidateUserPlaylists(user)
}

func (s *SpotifyService) userPlaylists(user string) ([]music.PlaylistSummary, error) {
	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return nil, ErrNotConnected
	}

	pls, err := GetPlaylists(ctx, sess, user, false)
	if err != nil {
		return nil, fmt.Errorf("get playlists: %w", err)
	}

	out := make([]music.PlaylistSummary, 0, len(pls)+1)
	if strings.EqualFold(user, sess.Username()) {
		out = append(out, likedSummary(user))
	}
	for _, p := range pls {
		id := strings.TrimPrefix(p.URI, URIPlaylistPrefix)
		if id == "" {
			continue
		}
		out = append(out, music.PlaylistSummary{
			Id:    id,
			Name:  p.Name,
			Cover: p.CoverURL,
			Owner: p.Owner,
		})
	}
	return out, nil
}

// Liked Songs is a user-scoped context, not a playlist, so it never shows up in
// the rootlist. It gets a synthetic summary whose id doubles as the context URI
// the player resolves.
func likedSummary(user string) music.PlaylistSummary {
	return music.PlaylistSummary{Id: "spotify:user:" + user + URICollectionSuffix}
}

func (s *SpotifyService) Search(query string, searchType string) ([]music.SearchResult, error) {
	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return nil, ErrNotConnected
	}
	return Search(ctx, sess, query, searchType)
}

func trackToSong(t Track) music.Song {
	id := strings.TrimPrefix(t.URI, URITrackPrefix)

	artists := make([]music.Artist, 0, len(t.Artists))
	for i, a := range t.Artists {
		if a == "" {
			continue
		}
		artist := music.Artist{Name: a}
		if i < len(t.ArtistURIs) {
			artist.Id = strings.TrimPrefix(t.ArtistURIs[i], URIArtistPrefix)
		}
		artists = append(artists, artist)
	}

	covers := []string{}
	if t.CoverURL != "" {
		covers = append(covers, t.CoverURL)
	}

	albumID := ""
	if strings.HasPrefix(t.AlbumURI, URIAlbumPrefix) {
		albumID = strings.TrimPrefix(t.AlbumURI, URIAlbumPrefix)
	}

	return music.Song{
		Id:       id,
		Url:      t.URI,
		Name:     t.Name,
		Duration: t.Duration / 1000,
		Share: music.Share{
			Id:  id,
			Url: "https://open.spotify.com/track/" + id,
		},
		Album: music.Album{
			Id:     albumID,
			Title:  t.Album,
			Url:    t.AlbumURI,
			Covers: covers,
		},
		Artists: artists,
	}
}

func (s *SpotifyService) GetArtist(uri string) (music.ArtistInfo, error) {
	normalizedURI := normalizeID(uri, "artist")
	return kv.Cached(musicBucket, "artist:"+normalizedURI, func() (music.ArtistInfo, error) {
		return s.artist(normalizedURI)
	})
}

func (s *SpotifyService) artist(normalizedURI string) (music.ArtistInfo, error) {
	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return music.ArtistInfo{}, ErrNotConnected
	}

	info, err := GetArtist(ctx, sess, normalizedURI)
	if err != nil {
		return music.ArtistInfo{}, err
	}

	songs := make([]music.Song, 0, len(info.Tracks))
	for _, t := range info.Tracks {
		songs = append(songs, trackToSong(t))
	}

	return music.ArtistInfo{
		Name:     info.Name,
		URI:      info.URI,
		ImageURL: info.ImageURL,
		Tracks:   songs,
	}, nil
}

func (s *SpotifyService) GetArtistDiscography(uri string) (music.ArtistDiscography, error) {
	normalizedURI := normalizeID(uri, "artist")
	return kv.Cached(musicBucket, "artist-discography:"+normalizedURI, func() (music.ArtistDiscography, error) {
		return s.artistDiscography(normalizedURI)
	})
}

func (s *SpotifyService) artistDiscography(normalizedURI string) (music.ArtistDiscography, error) {
	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return music.ArtistDiscography{}, ErrNotConnected
	}

	disco, err := GetArtistDiscography(ctx, sess, normalizedURI)
	if err != nil {
		return music.ArtistDiscography{}, err
	}

	albums := make([]music.AlbumSummary, 0, len(disco.Albums))
	for _, a := range disco.Albums {
		id := strings.TrimPrefix(a.URI, URIAlbumPrefix)
		if id == "" {
			continue
		}
		albums = append(albums, music.AlbumSummary{
			Id:    id,
			Name:  a.Name,
			Year:  a.Year,
			Cover: a.CoverURL,
			Group: a.Group,
		})
	}

	return music.ArtistDiscography{
		Name:   disco.Name,
		URI:    disco.URI,
		Albums: albums,
	}, nil
}

func (s *SpotifyService) GetAlbumTracks(uri string) ([]music.Song, error) {
	normalizedURI := normalizeID(uri, "album")
	return kv.Cached(musicBucket, "album:"+normalizedURI, func() ([]music.Song, error) {
		return s.albumTracks(normalizedURI)
	})
}

func (s *SpotifyService) albumTracks(normalizedURI string) ([]music.Song, error) {
	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return nil, ErrNotConnected
	}

	tracks, err := GetAlbumTracks(ctx, sess, normalizedURI)
	if err != nil {
		return nil, err
	}

	songs := make([]music.Song, 0, len(tracks))
	for _, t := range tracks {
		songs = append(songs, trackToSong(t))
	}

	return songs, nil
}
