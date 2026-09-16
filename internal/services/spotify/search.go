package spotify

import (
	"context"
	"fmt"
	"strings"

	librespot "github.com/devgianlu/go-librespot"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
	"github.com/devgianlu/go-librespot/session"
)

type SearchResult struct {
	URI        string
	Name       string
	Type       string
	CoverURL   string
	Artists    []string
	OwnerName  string
	TrackCount int32
}

func normalizeID(id, typ string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "spotify:") {
		return id
	}
	return "spotify:" + typ + ":" + id
}

func Search(ctx context.Context, sess *session.Session, query string, searchType string) ([]SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	switch searchType {
	case "user":
		return searchUser(ctx, sess, query)
	case "track":
		uri := normalizeID(query, "track")
		id, err := librespot.SpotifyIdFromUri(uri)
		if err != nil {
			return nil, fmt.Errorf("invalid track id: %w", err)
		}
		return getTrackResult(ctx, sess, *id)
	case "album":
		uri := normalizeID(query, "album")
		id, err := librespot.SpotifyIdFromUri(uri)
		if err != nil {
			return nil, fmt.Errorf("invalid album id: %w", err)
		}
		return getAlbumResult(ctx, sess, *id)
	case "artist":
		uri := normalizeID(query, "artist")
		id, err := librespot.SpotifyIdFromUri(uri)
		if err != nil {
			return nil, fmt.Errorf("invalid artist id: %w", err)
		}
		return getArtistResult(ctx, sess, *id)
	case "playlist":
		uri := normalizeID(query, "playlist")
		_, err := librespot.SpotifyIdFromUri(uri)
		if err != nil {
			return nil, fmt.Errorf("invalid playlist id: %w", err)
		}
		return getPlaylistResult(ctx, sess, uri)
	default:
		return nil, fmt.Errorf("invalid search type: %s", searchType)
	}
}

func searchUser(ctx context.Context, sess *session.Session, username string) ([]SearchResult, error) {
	pls, err := GetPlaylists(ctx, sess, username, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get user playlists: %w", err)
	}

	results := make([]SearchResult, 0, len(pls))
	for _, p := range pls {
		id := strings.TrimPrefix(p.URI, URIPlaylistPrefix)
		if id == "" {
			continue
		}
		results = append(results, SearchResult{
			URI:        p.URI,
			Name:       p.Name,
			Type:       "playlist",
			CoverURL:   p.CoverURL,
			OwnerName:  p.Owner,
			TrackCount: p.TrackCount,
		})
	}
	return results, nil
}

func getTrackResult(ctx context.Context, sess *session.Session, id librespot.SpotifyId) ([]SearchResult, error) {
	var track metadatapb.Track
	if err := sess.Spclient().ExtendedMetadataSimple(ctx, id, extmetadatapb.ExtensionKind_TRACK_V4, &track); err != nil {
		return nil, fmt.Errorf("failed to fetch track: %w", err)
	}
	t := trackFromProto(&track)
	return []SearchResult{{
		URI:      t.URI,
		Name:     t.Name,
		Type:     "track",
		CoverURL: t.CoverURL,
		Artists:  t.Artists,
	}}, nil
}

func getAlbumResult(ctx context.Context, sess *session.Session, id librespot.SpotifyId) ([]SearchResult, error) {
	var album metadatapb.Album
	if err := sess.Spclient().ExtendedMetadataSimple(ctx, id, extmetadatapb.ExtensionKind_ALBUM_V4, &album); err != nil {
		return nil, fmt.Errorf("failed to fetch album: %w", err)
	}

	result := SearchResult{
		URI:      id.Uri(),
		Name:     album.GetName(),
		Type:     "album",
		CoverURL: coverURLFromAlbum(&album),
	}
	for _, a := range album.GetArtist() {
		if name := a.GetName(); name != "" {
			result.Artists = append(result.Artists, name)
		}
	}
	return []SearchResult{result}, nil
}

func getArtistResult(ctx context.Context, sess *session.Session, id librespot.SpotifyId) ([]SearchResult, error) {
	var artist metadatapb.Artist
	if err := sess.Spclient().ExtendedMetadataSimple(ctx, id, extmetadatapb.ExtensionKind_ARTIST_V4, &artist); err != nil {
		return nil, fmt.Errorf("failed to fetch artist: %w", err)
	}

	return []SearchResult{{
		URI:  id.Uri(),
		Name: artist.GetName(),
		Type: "artist",
	}}, nil
}

func getPlaylistResult(ctx context.Context, sess *session.Session, uri string) ([]SearchResult, error) {
	meta, err := getPlaylistMetadata(ctx, sess, uri)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch playlist: %w", err)
	}
	return []SearchResult{{
		URI:        uri,
		Name:       meta.name,
		Type:       "playlist",
		CoverURL:   meta.cover,
		TrackCount: meta.trackCount,
	}}, nil
}
