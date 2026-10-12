package youtube

import (
	"errors"
	"strings"

	"github.com/Jota-Music/jota/internal/music"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// GetAudio returns the audio stream for a known YouTube video ID.
func (s *Service) GetAudio(id string) (music.Audio, error) {
	id = strings.TrimPrefix(id, music.YouTubePrefix)
	if !isYoutubeId(id) {
		return music.Audio{}, errors.New("invalid youtube id")
	}
	audio, err := fetchAudio(id)
	if err != nil {
		return music.Audio{}, err
	}
	return *audio, nil
}

// ResolveAudio returns a stream for any song. When the song already carries a
// linked YouTube ID it is used directly; if that video is unavailable it falls
// back to searching by the song's own metadata.
func (s *Service) ResolveAudio(song music.Song) (music.Audio, error) {
	if song.YoutubeId != "" {
		audio, err := s.GetAudio(song.YoutubeId)
		if err == nil {
			return audio, nil
		}
	}
	audio, err := s.searchAudio(song, audioQuery(song))
	if err == nil {
		return audio, nil
	}

	// A title-first query can come back empty; YouTube search indexes "artist
	// title" better, so retry the other way before giving up.
	if alt := audioQueryArtistFirst(song); alt != audioQuery(song) {
		if retry, retryErr := s.searchAudio(song, alt); retryErr == nil {
			return retry, nil
		}
	}

	return audio, err
}

func (s *Service) searchAudio(song music.Song, search string) (music.Audio, error) {
	cacheKey := song.Id
	if cached := cachedAudioBySong(cacheKey); cached != nil {
		return *cached, nil
	}

	// The linked id is the video an earlier resolve settled on — the user's
	// manual pick or the previous automatic one, and ranking must not demote it
	// behind a fresh pick. Playing it first skips the search, the slowest step of
	// a resolve; searching for alternatives is only worth it once it fails.
	var lastErr error
	tried := ""
	if id := s.YoutubeId(cacheKey); id != "" {
		tried = id
		audio, err := s.play(cacheKey, id)
		if err == nil {
			return *audio, nil
		}
		lastErr = err
	}

	videos, err := s.candidates(cacheKey, search, song)
	if err != nil {
		return music.Audio{}, err
	}

	for _, v := range videos {
		if v.ID == tried {
			continue
		}
		audio, err := s.play(cacheKey, v.ID)
		if err != nil {
			lastErr = err
			continue
		}
		return *audio, nil
	}

	return music.Audio{}, lastErr
}

// play resolves one candidate and records it as the song's source, so the next
// resolve skips the search. fetchAudio already serves the cached stream when it
// is still valid, so a separate cache read here would only probe googlevideo
// twice for the same URL.
func (s *Service) play(cacheKey, videoID string) (*music.Audio, error) {
	audio, err := fetchAudio(videoID)
	if err != nil {
		return nil, err
	}
	if cacheKey != "" {
		_ = s.SetYoutubeId(cacheKey, videoID)
		saveAudioBySong(cacheKey, *audio)
	}
	return audio, nil
}

// YoutubeId returns the YouTube video linked to a song, or "" when none.
func (s *Service) YoutubeId(cacheKey string) string {
	if v, err := youtubeSourceBucket.GetString(cacheKey); err == nil {
		return v
	}
	return ""
}

func (s *Service) SetYoutubeId(cacheKey, youtubeId string) error {
	// Invalidate cached audio so the new video is resolved on the next request.
	if old, err := youtubeSourceBucket.GetString(cacheKey); err == nil && old != "" && old != youtubeId {
		_ = audioBucket.Delete(old)
	}
	_ = audioBucket.Delete("spotify:" + cacheKey)
	return youtubeSourceBucket.SetString(cacheKey, youtubeId)
}

// audioQuery asks YouTube for the song by title, artist and album. The album
// disambiguates a title YouTube spells differently — "Lo que quieras" against
// Spotify's "Lo que quieres" — because the distributor keeps the release name in
// the upload's metadata. audioQueryArtistFirst stays album-free as the broader
// fallback, so an album that over-constrains the search can still recover.
func audioQuery(song music.Song) string {
	query := song.Name + " " + artistNames(song)
	if album := strings.TrimSpace(song.Album.Title); album != "" {
		query += " " + album
	}
	return strings.TrimSpace(query)
}

func audioQueryArtistFirst(song music.Song) string {
	names := artistNames(song)
	if names == "" {
		return strings.TrimSpace(song.Name)
	}
	return strings.TrimSpace(names + " " + song.Name)
}

func artistNames(song music.Song) string {
	names := make([]string, 0, len(song.Artists))
	for _, artist := range song.Artists {
		if artist.Name != "" {
			names = append(names, artist.Name)
		}
	}
	return strings.Join(names, ", ")
}

func isYoutubeId(id string) bool {
	return videoIdPattern.MatchString(id)
}
