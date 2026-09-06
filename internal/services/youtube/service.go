package youtube

import (
	"errors"
	"regexp"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetAudio(spotifyId, search string) (Audio, error) {
	if cached := GetCachedAudioBySpotifyId(spotifyId); cached != nil {
		return *cached, nil
	}

	youtubeId := spotifyId
	if search != "" {
		var err error
		youtubeId, err = GetSong(spotifyId, search)
		if err != nil {
			return Audio{}, err
		}
	} else if !isYoutubeId(spotifyId) {
		return Audio{}, errors.New("no search query and not a youtube id")
	}

	if cached := GetCachedAudioByYoutubeId(youtubeId); cached != nil {
		SaveAudioForSpotifyId(spotifyId, *cached, 0)
		return *cached, nil
	}

	audio, err := GetAudio(youtubeId)
	if err != nil {
		return Audio{}, err
	}

	SaveAudioForSpotifyId(spotifyId, *audio, 0)
	return *audio, nil
}

var youtubeIdRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

func isYoutubeId(id string) bool {
	return youtubeIdRegex.MatchString(id)
}

func (s *Service) SetYoutubeId(spotifyId, youtubeId string) error {
	return SetYoutubeId(spotifyId, youtubeId)
}
