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
	return bestImageURL(images)
}

// bestImageURL picks the smallest cover that still covers the largest UI tile
// (480px), falling back to the largest one when no image is big enough. The
// /__img proxy downscales from here to the exact display size, so this only
// needs to guarantee enough source resolution.
func bestImageURL(images []*metadatapb.Image) string {
	const targetW = 640

	var overW int32
	var over string
	var bestW int32
	var best string
	for _, img := range images {
		if img == nil || len(img.GetFileId()) == 0 {
			continue
		}
		id := hex.EncodeToString(img.GetFileId())
		if w := img.GetWidth(); w >= targetW && (overW == 0 || w < overW) {
			overW, over = w, id
		}
		if w := img.GetWidth(); w >= bestW {
			bestW, best = w, id
		}
	}
	if over == "" {
		return "https://i.scdn.co/image/" + best
	}
	return "https://i.scdn.co/image/" + over
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
}
