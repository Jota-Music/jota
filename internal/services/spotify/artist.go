package spotify

import (
	"context"
	"encoding/hex"

	librespot "github.com/devgianlu/go-librespot"
	"github.com/devgianlu/go-librespot/session"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
)

type ArtistInfo struct {
	Name     string
	URI      string
	ImageURL string
	Tracks   []Track
}

func portraitURLFromArtist(artist *metadatapb.Artist) string {
	images := artist.GetPortrait()
	if g := artist.GetPortraitGroup(); g != nil {
		images = append(images, g.GetImage()...)
	}
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

func GetArtist(ctx context.Context, sess *session.Session, uri string) (ArtistInfo, error) {
	id, err := spotifyID(uri, "artist")
	if err != nil {
		return ArtistInfo{}, err
	}
	var artist metadatapb.Artist
	if err := sess.Spclient().ExtendedMetadataSimple(ctx, id, extmetadatapb.ExtensionKind_ARTIST_V4, &artist); err != nil {
		return ArtistInfo{}, err
	}
	tracks := artistTopTracks(&artist)
	enrichTracks(ctx, sess, tracks)
	return ArtistInfo{
		Name:     artist.GetName(),
		URI:      uri,
		ImageURL: portraitURLFromArtist(&artist),
		Tracks:   tracks,
	}, nil
}

func GetArtistTopTracks(ctx context.Context, sess *session.Session, uri string) ([]Track, error) {
	info, err := GetArtist(ctx, sess, uri)
	if err != nil {
		return nil, err
	}
	return info.Tracks, nil
}

type ArtistDiscography struct {
	Name   string
	URI    string
	Albums []AlbumRef
}

func GetArtistDiscography(ctx context.Context, sess *session.Session, uri string) (ArtistDiscography, error) {
	id, err := spotifyID(uri, "artist")
	if err != nil {
		return ArtistDiscography{}, err
	}
	var artist metadatapb.Artist
	if err := sess.Spclient().ExtendedMetadataSimple(ctx, id, extmetadatapb.ExtensionKind_ARTIST_V4, &artist); err != nil {
		return ArtistDiscography{}, err
	}
	albums := albumRefsFromArtist(&artist)
	enrichAlbums(ctx, sess, albums)
	return ArtistDiscography{
		Name:   artist.GetName(),
		URI:    uri,
		Albums: albums,
	}, nil
}

func GetArtistAlbums(ctx context.Context, sess *session.Session, uri string) ([]AlbumRef, error) {
	info, err := GetArtistDiscography(ctx, sess, uri)
	return info.Albums, err
}

func artistTopTracks(artist *metadatapb.Artist) []Track {
	for _, group := range artist.GetTopTrack() {
		var tracks []Track
		for _, tr := range group.GetTrack() {
			if tr != nil {
				tracks = append(tracks, trackFromProto(tr))
			}
		}
		if len(tracks) > 0 {
			return tracks
		}
	}
	return nil
}

func albumRefsFromArtist(artist *metadatapb.Artist) []AlbumRef {
	sections := []struct {
		label string
		group []*metadatapb.AlbumGroup
	}{
		{"álbum", artist.GetAlbumGroup()},
		{"single", artist.GetSingleGroup()},
		{"compilación", artist.GetCompilationGroup()},
	}
	var out []AlbumRef
	for _, s := range sections {
		for _, g := range s.group {
			if g == nil {
				continue
			}
			for _, a := range g.GetAlbum() {
				if a == nil || len(a.GetGid()) != 16 {
					continue
				}
				out = append(out, AlbumRef{
					URI:   librespot.SpotifyIdFromGid(librespot.SpotifyIdTypeAlbum, a.GetGid()).Uri(),
					Year:  a.GetDate().GetYear(),
					Group: s.label,
				})
			}
		}
	}
	return out
}
