package youtube

import "regexp"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetAudio(spotifyId, search string) (Audio, error) {
	if cached := GetCachedAudioBySpotifyId(spotifyId); cached != nil {
		return *cached, nil
	}

	ids, err := candidates(spotifyId, search)
	if err != nil {
		return Audio{}, err
	}

	var lastErr error
	for _, youtubeId := range ids {
		audio := GetCachedAudioByYoutubeId(youtubeId)
		if audio == nil {
			audio, err = GetAudio(youtubeId)
			if err != nil {
				lastErr = err
				continue
			}
		}

		if search != "" {
			_ = SetYoutubeId(spotifyId, youtubeId)
		}
		SaveAudioForSpotifyId(spotifyId, *audio, 0)
		return *audio, nil
	}

	return Audio{}, lastErr
}

var youtubeIdRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

func isYoutubeId(id string) bool {
	return youtubeIdRegex.MatchString(id)
}

func (s *Service) SetYoutubeId(spotifyId, youtubeId string) error {
	return SetYoutubeId(spotifyId, youtubeId)
}
