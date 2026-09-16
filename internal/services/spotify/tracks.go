package spotify

import (
	"context"
	"strings"
	"sync"

	librespot "github.com/devgianlu/go-librespot"
	connectpb "github.com/devgianlu/go-librespot/proto/spotify/connectstate"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
	"github.com/devgianlu/go-librespot/session"
)

type Track struct {
	URI         string
	Name        string
	Artists     []string
	ArtistURIs  []string
	Album       string
	AlbumURI    string
	CoverURL    string
	Duration    int
	TrackNumber int
	Popularity  int
}

func trackFromProto(track *metadatapb.Track) Track {
	t := Track{}
	if len(track.GetGid()) == 16 {
		t.URI = librespot.SpotifyIdFromGid(librespot.SpotifyIdTypeTrack, track.GetGid()).Uri()
	}
	mergeTrackFromProto(&t, track)
	return t
}

func mergeTrackFromProto(t *Track, track *metadatapb.Track) {
	if t.Name == "" {
		t.Name = track.GetName()
	}
	if len(t.Artists) == 0 && len(t.ArtistURIs) == 0 {
		for _, a := range track.GetArtist() {
			if name := a.GetName(); name != "" {
				t.Artists = append(t.Artists, name)
			}
			if len(a.GetGid()) == 16 {
				t.ArtistURIs = append(t.ArtistURIs, librespot.SpotifyIdFromGid(librespot.SpotifyIdTypeArtist, a.GetGid()).Uri())
			}
		}
	}
	if album := track.GetAlbum(); album != nil {
		if t.Album == "" {
			t.Album = album.GetName()
		}
		if t.AlbumURI == "" && len(album.GetGid()) == 16 {
			t.AlbumURI = librespot.SpotifyIdFromGid(librespot.SpotifyIdTypeAlbum, album.GetGid()).Uri()
		}
		if t.CoverURL == "" {
			t.CoverURL = coverURLFromAlbum(album)
		}
	}
	if t.Duration == 0 {
		t.Duration = int(track.GetDuration())
	}
	if t.TrackNumber == 0 {
		t.TrackNumber = int(track.GetNumber())
	}
	if t.Popularity == 0 {
		t.Popularity = int(track.GetPopularity())
	}
}

func tracksFromContext(cts []*connectpb.ContextTrack) []Track {
	out := make([]Track, 0, len(cts))
	for _, ct := range cts {
		u := ct.GetUri()
		if u == "" || strings.HasPrefix(u, URILocalPrefix) {
			continue
		}
		out = append(out, Track{URI: u})
	}
	return out
}

func enrichTracks(ctx context.Context, sess *session.Session, tracks []Track) {
	const (
		batchSize = 50
		maxConc   = 4
	)

	uris, byURI := groupByURI(tracks)
	if len(uris) == 0 {
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConc)
	for start := 0; start < len(uris); start += batchSize {
		end := start + batchSize
		if end > len(uris) {
			end = len(uris)
		}
		batch := uris[start:end]
		wg.Add(1)
		sem <- struct{}{}
		go func(us []string) {
			defer wg.Done()
			defer func() { <-sem }()
			enrichBatch(ctx, sess, tracks, byURI, us)
		}(batch)
	}
	wg.Wait()

	var missing []int
	for i := range tracks {
		if needsEnrich(tracks[i]) {
			missing = append(missing, i)
		}
	}
	if len(missing) == 0 {
		return
	}

	sem = make(chan struct{}, DefaultEnrichConcurrency)
	for _, i := range missing {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			enrichOneTrack(ctx, sess, &tracks[idx])
		}(i)
	}
	wg.Wait()
}

func needsEnrich(t Track) bool {
	return t.Name == "" || t.CoverURL == "" || len(t.Artists) == 0
}

// groupByURI maps every index that still needs metadata to its track URI and
// returns each unique URI once, so repeated URIs (common in large playlists)
// share a single lookup instead of leaving duplicates unenriched for the
// one-by-one fallback.
func groupByURI(tracks []Track) ([]string, map[string][]int) {
	uris := make([]string, 0, len(tracks))
	index := make(map[string][]int, len(tracks))
	for i := range tracks {
		if !needsEnrich(tracks[i]) {
			continue
		}
		uri := tracks[i].URI
		if _, err := librespot.SpotifyIdFromUri(uri); err != nil {
			continue
		}
		if _, seen := index[uri]; !seen {
			uris = append(uris, uri)
		}
		index[uri] = append(index[uri], i)
	}
	return uris, index
}

func enrichBatch(ctx context.Context, sess *session.Session, tracks []Track, byURI map[string][]int, uris []string) {
	if len(uris) == 0 {
		return
	}

	req := &extmetadatapb.BatchedEntityRequest{}
	for _, uri := range uris {
		req.EntityRequest = append(req.EntityRequest, &extmetadatapb.EntityRequest{
			EntityUri: uri,
			Query: []*extmetadatapb.ExtensionQuery{{
				ExtensionKind: extmetadatapb.ExtensionKind_TRACK_V4,
			}},
		})
	}

	resp, err := sess.Spclient().ExtendedMetadata(ctx, req)
	if err != nil {
		return
	}
	for _, arr := range resp.ExtendedMetadata {
		if arr.ExtensionKind != extmetadatapb.ExtensionKind_TRACK_V4 {
			continue
		}
		for _, data := range arr.ExtensionData {
			matches, ok := byURI[data.EntityUri]
			if !ok || data.GetHeader().GetStatusCode() != 200 || data.ExtensionData == nil {
				continue
			}
			var track metadatapb.Track
			if err := data.ExtensionData.UnmarshalTo(&track); err != nil {
				continue
			}
			for _, i := range matches {
				mergeTrackFromProto(&tracks[i], &track)
			}
		}
	}
}

func enrichOneTrack(ctx context.Context, sess *session.Session, t *Track) {
	id, err := librespot.SpotifyIdFromUri(t.URI)
	if err != nil {
		return
	}
	var track metadatapb.Track
	if err := sess.Spclient().ExtendedMetadataSimple(ctx, *id, extmetadatapb.ExtensionKind_TRACK_V4, &track); err != nil {
		return
	}
	mergeTrackFromProto(t, &track)
}
