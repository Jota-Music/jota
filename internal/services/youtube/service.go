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

	videos, err := s.candidates(cacheKey, search, song)
	if err != nil {
		return music.Audio{}, err
	}

	var lastErr error
	for _, v := range videos {
		audio := cachedAudio(v.ID)
		if audio == nil {
			audio, err = fetchAudio(v.ID)
			if err != nil {
				lastErr = err
				continue
			}
		}

		if search != "" {
			_ = s.SetYoutubeId(cacheKey, v.ID)
		}
		saveAudioBySong(cacheKey, *audio)
		return *audio, nil
	}

	return music.Audio{}, lastErr
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

func audioQuery(song music.Song) string {
	return strings.TrimSpace(song.Name + " " + artistNames(song))
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
