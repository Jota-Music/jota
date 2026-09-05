package spotify

import (
	"context"
	"fmt"
	"strings"

	"jota/server/internal/music"

	"github.com/devgianlu/go-librespot/session"
)

func (s *SpotifyService) GetSong(id string) (music.Song, error) {
	uri := normalizeTrackID(id)

	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return music.Song{}, ErrNotConnected
	}

	track, err := GetTrack(ctx, sess, uri)
	if err != nil {
		return music.Song{}, err
	}
	return trackToSong(track), nil
}

func (s *SpotifyService) GetPlaylist(playlistID string, page, size int) (music.Playlist, error) {
	if page < 0 {
		page = 0
	}
	if size <= 0 {
		size = 20
	}

	uri := normalizePlaylistID(playlistID)
	offset := page * size

	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return music.Playlist{}, ErrNotConnected
	}

	uris, err := resolveContextURIs(ctx, sess, uri)
	if err != nil {
		return music.Playlist{}, fmt.Errorf("resolve playlist: %w", err)
	}
	total := len(uris)

	tracks, err := getTracksFromURIs(ctx, sess, uri, offset, size)
	if err != nil {
		return music.Playlist{}, err
	}

	songs := make([]music.Song, 0, len(tracks))
	for _, t := range tracks {
		songs = append(songs, trackToSong(t))
	}

	return music.Playlist{
		Songs: songs,
		Page: music.Page{
			Size:    size,
			Offset:  offset,
			Total:   total,
			HasNext: offset+size < total,
		},
	}, nil
}

func getTracksFromURIs(ctx context.Context, sess *session.Session, uri string, offset, limit int) ([]Track, error) {
	_, err := spotifyID(uri, "playlist")
	if err != nil {
		return nil, err
	}
	uris, err := resolveContextURIs(ctx, sess, uri)
	if err != nil {
		return nil, err
	}
	if offset >= len(uris) {
		return []Track{}, nil
	}
	end := len(uris)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	out := make([]Track, 0, end-offset)
	for _, u := range uris[offset:end] {
		if strings.HasPrefix(u, URILocalPrefix) {
			continue
		}
		out = append(out, Track{URI: u})
	}
	enrichTracks(ctx, sess, out)
	return out, nil
}

func (s *SpotifyService) GetUserPlaylists(user string) ([]music.PlaylistSummary, error) {
	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return nil, ErrNotConnected
	}

	pls, err := GetPlaylists(ctx, sess, user, false)
	if err != nil {
		return nil, fmt.Errorf("get playlists: %w", err)
	}

	out := make([]music.PlaylistSummary, 0, len(pls))
	for _, p := range pls {
		id := strings.TrimPrefix(p.URI, URIPlaylistPrefix)
		if id == "" {
			continue
		}
		out = append(out, music.PlaylistSummary{
			Id:     id,
			Name:   p.Name,
			Mosaic: "",
			Cover:  p.CoverURL,
		})
	}
	return out, nil
}

func (s *SpotifyService) Search(query string, searchType string) ([]music.SearchResult, error) {
	ctx := context.Background()
	sess := s.Session()
	if sess == nil {
		return nil, ErrNotConnected
	}

	results, err := Search(ctx, sess, query, searchType)
	if err != nil {
		return nil, err
	}

	out := make([]music.SearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, music.SearchResult{
			URI:        r.URI,
			Name:       r.Name,
			Type:       r.Type,
			CoverURL:   r.CoverURL,
			Artists:    r.Artists,
			OwnerName:  r.OwnerName,
			TrackCount: r.TrackCount,
		})
	}
	return out, nil
}

func normalizeTrackID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, URITrackPrefix) {
		return id
	}
	return URITrackPrefix + id
}

func normalizePlaylistID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, URIPlaylistPrefix) {
		return id
	}
	return URIPlaylistPrefix + id
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
			Title:  t.Album,
			Url:    t.AlbumURI,
			Covers: covers,
		},
		Artists: artists,
	}
}

func normalizeArtistID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, URIArtistPrefix) {
		return id
	}
	return URIArtistPrefix + id
}

func normalizeAlbumID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, URIAlbumPrefix) {
		return id
	}
	return URIAlbumPrefix + id
}

func (s *SpotifyService) GetArtist(uri string) (music.ArtistInfo, error) {
	normalizedURI := normalizeArtistID(uri)
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
	normalizedURI := normalizeArtistID(uri)
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
	normalizedURI := normalizeAlbumID(uri)
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
