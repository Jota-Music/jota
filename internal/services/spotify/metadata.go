package spotify

import (
	"context"
	"strings"

	"github.com/devgianlu/go-librespot/session"
)

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
