package spotify

import (
	"context"
	"fmt"
	"strings"

	"github.com/devgianlu/go-librespot/session"

	"github.com/Jota-Music/jota/internal/music"
)

// GetSongRadio resolves the song-station context for a track and returns its
// tracks as a synthetic playlist. Radio is dynamic by nature, so it is never
// cached: every call asks the context servers for a fresh set of pages.
func (s *SpotifyService) GetSongRadio(id string) (music.Playlist, error) {
	trackID := seed(id)

	sess := s.Session()
	if sess == nil {
		return music.Playlist{}, ErrNotConnected
	}

	ctx := context.Background()

	// Modern clients drive song radio through station contexts; the legacy
	// radio form is the fallback when a comptible station is unavailable.
	tracks, err := s.radioTracks(ctx, sess, URIStationPrefix+trackID)
	if err != nil {
		tracks, err = s.radioTracks(ctx, sess, URIRadioPrefix+trackID)
		if err != nil {
			return music.Playlist{}, err
		}
	}

	songs := make([]music.Song, 0, len(tracks))
	for _, t := range tracks {
		songs = append(songs, trackToSong(t))
	}

	name := "Radio"
	if seed, err := s.GetSong(trackID); err == nil && seed.Name != "" {
		name = "Radio · " + seed.Name
	}

	cover := ""
	if len(songs) > 0 {
		if covers := songs[0].Album.Covers; len(covers) > 0 {
			cover = covers[0]
		}
	}

	return music.Playlist{Name: name, Cover: cover, Songs: songs}, nil
}

// seed is the track id a radio is built around, whichever form the caller sent.
// The radio page prefixes the song id it was given, and a Spotify song id is
// already "spotify:<id>", so the inner form has to be normalized too.
func seed(id string) string {
	id = strings.TrimPrefix(strings.TrimPrefix(id, URIRadioPrefix), URIStationPrefix)
	return strings.TrimPrefix(normalizeID(id, "track"), URITrackPrefix)
}

func (s *SpotifyService) radioTracks(ctx context.Context, sess *session.Session, uri string) ([]Track, error) {
	ctxTracks, err := resolveRadioTracks(ctx, sess, uri)
	if err != nil {
		return nil, fmt.Errorf("resolve radio %s: %w", uri, err)
	}
	tracks := tracksFromContext(ctxTracks)
	enrichTracks(ctx, sess, tracks)
	return tracks, nil
}
