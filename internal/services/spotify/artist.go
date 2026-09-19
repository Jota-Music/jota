package spotify

import (
	"context"

	librespot "github.com/devgianlu/go-librespot"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"
	"github.com/devgianlu/go-librespot/session"
)

func portraitURLFromArtist(artist *metadatapb.Artist) string {
	images := artist.GetPortrait()
	if g := artist.GetPortraitGroup(); g != nil {
		images = append(images, g.GetImage()...)
	}
	return bestImageURL(images)
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
		{"album", artist.GetAlbumGroup()},
		{"single", artist.GetSingleGroup()},
		{"compilation", artist.GetCompilationGroup()},
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
