package youtube

import (
	"errors"
	"regexp"
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
	return s.searchAudio(song.Id, audioQuery(song))
}

func (s *Service) searchAudio(cacheKey, search string) (music.Audio, error) {
	if cached := cachedAudioBySong(cacheKey); cached != nil {
		return *cached, nil
	}

	ids, err := s.candidates(cacheKey, search)
	if err != nil {
		return music.Audio{}, err
	}

	var lastErr error
	for _, youtubeId := range ids {
		audio := cachedAudioByYoutube(youtubeId)
		if audio == nil {
			audio, err = fetchAudio(youtubeId)
			if err != nil {
				lastErr = err
				continue
			}
		}

		if search != "" {
			_ = s.SetYoutubeId(cacheKey, youtubeId)
		}
		saveAudioBySong(cacheKey, *audio)
		return *audio, nil
	}

	return music.Audio{}, lastErr
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
	names := make([]string, 0, len(song.Artists))
	for _, artist := range song.Artists {
		if artist.Name != "" {
			names = append(names, artist.Name)
		}
	}
	return strings.TrimSpace(song.Name + " " + strings.Join(names, ", "))
}

var youtubeIdRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

func isYoutubeId(id string) bool {
	return youtubeIdRegex.MatchString(id)
}
