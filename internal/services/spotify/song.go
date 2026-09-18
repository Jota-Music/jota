package spotify

import (
	"context"

	librespot "github.com/devgianlu/go-librespot"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"

	"github.com/Jota-Music/jota/internal/music"
)

// GetSong resolves a single track by id (bare id, URI or link target) to its
// full metadata.
func (s *SpotifyService) GetSong(id string) (music.Song, error) {
	uri := normalizeID(id, "track")
	return cachedTrack(uri, func() (music.Song, error) {
		return s.song(uri)
	})
}

func (s *SpotifyService) song(uri string) (music.Song, error) {
	sess := s.Session()
	if sess == nil {
		return music.Song{}, ErrNotConnected
	}

	sid, err := librespot.SpotifyIdFromUri(uri)
	if err != nil {
		return music.Song{}, err
	}

	var track metadatapb.Track
	if err := sess.Spclient().ExtendedMetadataSimple(context.Background(), *sid, extmetadatapb.ExtensionKind_TRACK_V4, &track); err != nil {
		return music.Song{}, err
	}

	return trackToSong(trackFromProto(&track)), nil
}
