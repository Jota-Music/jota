package spotify

import (
	"context"
	"fmt"
	"io"
	"strings"

	librespot "github.com/devgianlu/go-librespot"
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
	return fmt.Sprintf("se esperaba spotify:%s:..., se recibió spotify:%s: (%s)", e.Expected, e.Got, e.URI)
}

func resolveContextURIs(ctx context.Context, sess *session.Session, uri string) ([]string, error) {
	resolved, err := sess.Spclient().ContextResolve(ctx, uri)
	if err != nil {
		return nil, err
	}
	cr, err := spclient.NewContextResolver(ctx, &librespot.NullLogger{}, sess.Spclient(), resolved)
	if err != nil {
		return nil, err
	}

	var uris []string
	for page := 0; ; page++ {
		pageTracks, err := cr.Page(ctx, page)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		for _, ct := range pageTracks {
			u := ct.GetUri()
			if u == "" && len(ct.GetGid()) == 16 {
				u = librespot.SpotifyIdFromGid(cr.Type(), ct.GetGid()).Uri()
			}
			if u != "" {
				uris = append(uris, u)
			}
		}
	}
	return uris, nil
}
