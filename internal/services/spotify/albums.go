package spotify

import (
	"context"
	"encoding/hex"
	"sync"

	librespot "github.com/devgianlu/go-librespot"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
	"github.com/devgianlu/go-librespot/session"
)

type AlbumRef struct {
	URI      string
	Name     string
	Year     int32
	CoverURL string
	Tracks   []Track
	Group    string
}

func GetAlbumTracks(ctx context.Context, sess *session.Session, uri string) ([]Track, error) {
	id, err := spotifyID(uri, "album")
	if err != nil {
		return nil, err
	}
	var album metadatapb.Album
	if err := sess.Spclient().ExtendedMetadataSimple(ctx, id, extmetadatapb.ExtensionKind_ALBUM_V4, &album); err != nil {
		return nil, err
	}
	tracks := tracksFromAlbum(&album)
	enrichTracks(ctx, sess, tracks)
	return tracks, nil
}

func tracksFromAlbum(album *metadatapb.Album) []Track {
	var tracks []Track
	for _, disc := range album.GetDisc() {
		for _, tr := range disc.GetTrack() {
			if tr != nil {
				tracks = append(tracks, trackFromProto(tr))
			}
		}
	}
	return tracks
}

func coverURLFromAlbum(album *metadatapb.Album) string {
	images := append(album.GetCover(), album.GetCoverGroup().GetImage()...)
	var bestW int32
	var best string
	for _, img := range images {
		if img == nil || len(img.GetFileId()) == 0 {
			continue
		}
		if w := img.GetWidth(); w >= bestW {
			bestW = w
			best = hex.EncodeToString(img.GetFileId())
		}
	}
	if best == "" {
		return ""
	}
	return "https://i.scdn.co/image/" + best
}

func enrichAlbums(ctx context.Context, sess *session.Session, albums []AlbumRef) {
	sem := make(chan struct{}, AlbumEnrichConcurrency)
	var wg sync.WaitGroup
	for i := range albums {
		if albums[i].Name != "" && albums[i].CoverURL != "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			enrichOneAlbum(ctx, sess, &albums[idx])
		}(i)
	}
	wg.Wait()
}

func enrichOneAlbum(ctx context.Context, sess *session.Session, a *AlbumRef) {
	spotID, err := librespot.SpotifyIdFromUri(a.URI)
	if err != nil {
		return
	}
	var album metadatapb.Album
	if err := sess.Spclient().ExtendedMetadataSimple(ctx, *spotID, extmetadatapb.ExtensionKind_ALBUM_V4, &album); err != nil {
		return
	}
	if a.Name == "" {
		a.Name = album.GetName()
	}
	if a.Year == 0 {
		a.Year = album.GetDate().GetYear()
	}
	a.CoverURL = coverURLFromAlbum(&album)
	a.Tracks = tracksFromAlbum(&album)
	enrichTracks(ctx, sess, a.Tracks)
}
