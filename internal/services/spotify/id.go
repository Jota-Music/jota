package spotify

import (
	"context"
	"fmt"
	"io"
	"strings"

	librespot "github.com/devgianlu/go-librespot"
	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"
	"github.com/devgianlu/go-librespot/session"
	"github.com/devgianlu/go-librespot/spclient"
)

func spotifyID(uri, typ string) (librespot.SpotifyId, error) {
	id, err := librespot.SpotifyIdFromUri(uri)
	if err != nil {
		return librespot.SpotifyId{}, err
	}
	if !strings.HasPrefix(uri, "spotify:"+typ+":") {
		return librespot.SpotifyId{}, &TypeMismatchError{Expected: typ, Got: id.Type(), URI: uri}
	}
	return *id, nil
}

type TypeMismatchError struct {
	Expected string
	Got      librespot.SpotifyIdType
	URI      string
}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("expected spotify:%s:..., got spotify:%s: (%s)", e.Expected, e.Got, e.URI)
}

func resolveContextTracks(ctx context.Context, sess *session.Session, uri string) ([]*connectpb.ContextTrack, error) {
	resolved, err := sess.Spclient().ContextResolve(ctx, uri)
	if err != nil {
		return nil, err
	}
	cr, err := spclient.NewContextResolver(ctx, &librespot.NullLogger{}, sess.Spclient(), resolved)
	if err != nil {
		return nil, err
	}

	var tracks []*connectpb.ContextTrack
	for page := 0; ; page++ {
		pageTracks, err := cr.Page(ctx, page)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		for _, ct := range pageTracks {
			if ct.GetUri() == "" && len(ct.GetGid()) == 16 {
				ct.Uri = librespot.SpotifyIdFromGid(cr.Type(), ct.GetGid()).Uri()
			}
			if ct.GetUri() != "" {
				tracks = append(tracks, ct)
			}
		}
	}
	return tracks, nil
}

func resolveContextURIs(ctx context.Context, sess *session.Session, uri string) ([]string, error) {
	tracks, err := resolveContextTracks(ctx, sess, uri)
	if err != nil {
		return nil, err
	}
	uris := make([]string, 0, len(tracks))
	for _, ct := range tracks {
		uris = append(uris, ct.GetUri())
	}
	return uris, nil
}
