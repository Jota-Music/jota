package youtube

import (
	"errors"
	"fmt"
	"jota/server/internal/kv"
	"strings"
	"time"
)

var youtubeSourceBucket = kv.UseBucket("youtube-source")

type Audio struct {
	Url      string `json:"url"`
	Duration int    `json:"duration"`
	ExpireAt int64  `json:"expireAt"`
}

func GetAudio(youtubeId string) (*Audio, error) {
	youtubeId = strings.TrimSpace(youtubeId)
	if youtubeId == "" {
		return nil, fmt.Errorf("youtube id is empty")
	}

	var cached Audio
	err := youtubeSourceBucket.GetObject(youtubeId, &cached)
	if err == nil && audioCacheStillValid(&cached) {
		return &cached, nil
	}
	if err != nil && !errors.Is(err, kv.KeyNotFoundError) {
		return nil, fmt.Errorf("youtube audio cache read: %w", err)
	}

	info, err := getStreamURLViaBrowser(youtubeId)
	if err != nil {
		return nil, err
	}

	ttl := time.Duration(info.ExpiresAt-time.Now().Unix()-30) * time.Second
	if ttl <= 0 {
		return nil, errors.New("stream already expired")
	}

	audio := Audio{
		Url:      info.URL,
		Duration: 0,
		ExpireAt: info.ExpiresAt,
	}

	if err := youtubeSourceBucket.SetObject(youtubeId, audio, ttl); err != nil {
		return nil, fmt.Errorf("youtube audio cache write: %w", err)
	}

	return &audio, nil
}

func audioCacheStillValid(a *Audio) bool {
	if a == nil || a.Url == "" || a.ExpireAt <= 0 {
		return false
	}
	return a.ExpireAt > time.Now().Unix()+30
}

func SetYoutubeId(id string, youtubeId string) error {
	return youtubeSourceBucket.SetString(id, youtubeId)
}
