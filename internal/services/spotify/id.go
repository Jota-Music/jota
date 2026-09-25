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

func resolveContextTracks(ctx context.Context, sess *session.Session, uri string) ([]*connectpb.ContextTrack, error) {
	sp := sess.Spclient()
	resolved, err := sp.ContextResolve(ctx, uri)
	if err != nil {
		return nil, err
	}
	return contextTracks(ctx, sp, resolved)
}

func contextTracks(ctx context.Context, sp *spclient.Spclient, resolved *connectpb.Context) ([]*connectpb.ContextTrack, error) {
	cr, err := spclient.NewContextResolver(ctx, &librespot.NullLogger{}, sp, resolved)
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

// radioMaxPages bounds how far a radio context is walked. Stations paginate
// without end — every page yields tracks and names a next one, so there is no
// io.EOF to stop on. A radio is a listening session, not a finite catalog: a
// couple of pages is already a long session, so the loop stops there.
const radioMaxPages = 2

func resolveRadioTracks(ctx context.Context, sess *session.Session, uri string) (ctxTracks []*connectpb.ContextTrack, err error) {
	defer func() {
		if recover() != nil {
			// A station context arriving without usable pages would otherwise
			// panic the resolver out of the Go↔JS bridge.
			ctxTracks = nil
			err = fmt.Errorf("radio context %s has no readable pages", uri)
		}
	}()

	resolved, err := sess.Spclient().ContextResolve(ctx, uri)
	if err != nil {
		return nil, err
	}
	cr, err := spclient.NewContextResolver(ctx, &librespot.NullLogger{}, sess.Spclient(), resolved)
	if err != nil {
		return nil, err
	}

	for page := 0; page < radioMaxPages; page++ {
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
				ctxTracks = append(ctxTracks, ct)
			}
		}
	}
	return ctxTracks, nil
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
