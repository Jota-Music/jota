package spotify

import (
	"context"
	"sync"

	librespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/session"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
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

func enrichTracks(ctx context.Context, sess *session.Session, tracks []Track) {
	sem := make(chan struct{}, DefaultEnrichConcurrency)
	var wg sync.WaitGroup
	for i := range tracks {
		if tracks[i].Name != "" && tracks[i].CoverURL != "" {
			continue
		}
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

func enrichOneTrack(ctx context.Context, sess *session.Session, t *Track) {
	spotID, err := librespot.SpotifyIdFromUri(t.URI)
	if err != nil {
		return
	}
	var track metadatapb.Track
	if err := sess.Spclient().ExtendedMetadataSimple(ctx, *spotID, extmetadatapb.ExtensionKind_TRACK_V4, &track); err != nil {
		return
	}
	mergeTrackFromProto(t, &track)
}
