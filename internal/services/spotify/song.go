package spotify

import (
	"context"

	librespot "github.com/devgianlu/go-librespot"
	extmetadatapb "github.com/devgianlu/go-librespot/proto/spotify/extendedmetadata"
	metadatapb "github.com/devgianlu/go-librespot/proto/spotify/metadata"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

// GetSong resolves a single track by id (bare id, URI or link target) to its
// full metadata.
func (s *SpotifyService) GetSong(id string) (music.Song, error) {
	uri := normalizeID(id, "track")
	return kv.Cached(musicBucket, "track:"+uri, func() (music.Song, error) {
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

// CachedSong returns metadata previously stored for a track without hitting the
// network. It is used to keep a backup self-contained and to show titles even
// when Spotify is not connected. ok is false when nothing is cached.
func (s *SpotifyService) CachedSong(id string) (music.Song, bool) {
	uri := normalizeID(id, "track")
	type cachedTrack struct {
		Value       music.Song `json:"value"`
		Fingerprint string     `json:"fingerprint"`
	}
	var stored cachedTrack
	if err := musicBucket.GetObject("track:"+uri, &stored); err != nil || stored.Fingerprint == "" {
		return music.Song{}, false
	}
	return stored.Value, true
}

// SetCachedSong stores metadata for a track, e.g. when importing a backup, so
// later GetSong calls can resolve it without a network round trip.
func (s *SpotifyService) SetCachedSong(id string, song music.Song) {
	uri := normalizeID(id, "track")
	_ = musicBucket.SetObject("track:"+uri, struct {
		Value       music.Song `json:"value"`
		Fingerprint string     `json:"fingerprint"`
	}{Value: song, Fingerprint: "imported"})
}
