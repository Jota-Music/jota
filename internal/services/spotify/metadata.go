package spotify

import (
	"context"
	"strings"

	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
	"github.com/devgianlu/go-librespot/session"
)

func GetTrack(ctx context.Context, sess *session.Session, uri string) (Track, error) {
	id, err := spotifyID(uri, "track")
	if err != nil {
		return Track{}, err
	}
	var track metadatapb.Track
	if err := sess.Spclient().ExtendedMetadataSimple(ctx, id, extmetadatapb.ExtensionKind_TRACK_V4, &track); err != nil {
		return Track{}, err
	}
	return trackFromProto(&track), nil
}

func GetTracks(ctx context.Context, sess *session.Session, uri string) ([]Track, error) {
	switch {
	case strings.HasPrefix(uri, URIPlaylistPrefix):
		return GetPlaylistTracks(ctx, sess, uri)
	case strings.HasPrefix(uri, URIAlbumPrefix):
		return GetAlbumTracks(ctx, sess, uri)
	case strings.HasPrefix(uri, URIArtistPrefix):
		return GetArtistTopTracks(ctx, sess, uri)
	}
	return nil, &UnsupportedURIError{URI: uri}
}

type UnsupportedURIError struct {
	URI string
}

func (e *UnsupportedURIError) Error() string {
	return "uri no soportada: " + e.URI + " (soporta: playlist, album, artist)"
}

func GetPlaylistTracks(ctx context.Context, sess *session.Session, uri string) ([]Track, error) {
	return GetPlaylistTracksPage(ctx, sess, uri, 0, 0)
}

func GetPlaylistTracksPage(ctx context.Context, sess *session.Session, uri string, offset, limit int) ([]Track, error) {
	if _, err := spotifyID(uri, "playlist"); err != nil {
		return nil, err
	}
	uris, err := resolveContextURIs(ctx, sess, uri)
	if err != nil {
		return nil, err
	}
	if offset >= len(uris) {
		return nil, nil
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
