package youtube

import (
	"context"
	"errors"
	"fmt"
	"jota/server/internal/kv"
	"os/exec"
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

	streamURL, err := getStreamURL(youtubeId)
	if err != nil {
		return nil, err
	}

	ttl := time.Duration(streamURL.ExpiresAt-time.Now().Unix()-30) * time.Second
	if ttl <= 0 {
		return nil, errors.New("stream already expired")
	}

	audio := Audio{
		Url:      streamURL.URL,
		Duration: streamURL.Duration,
		ExpireAt: streamURL.ExpiresAt,
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

type StreamInfo struct {
	URL       string
	Duration  int
	ExpiresAt int64
}

func getStreamURL(youtubeId string) (*StreamInfo, error) {
	info, err := getStreamURLViaBrowser(youtubeId)
	if err == nil && info.URL != "" {
		return info, nil
	}

	return getStreamURLViaYTDLP(youtubeId)
}

func getStreamURLViaYTDLP(youtubeId string) (*StreamInfo, error) {
	bin, err := Ensure()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		bin,
		"--user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"-f", "bestaudio[ext=m4a]/bestaudio",
		"--print", "%(url)s",
		fmt.Sprintf("https://www.youtube.com/watch?v=%s", youtubeId),
	)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp: %s | %w", strings.TrimSpace(stderr.String()), err)
	}

	return &StreamInfo{URL: strings.TrimSpace(stdout.String())}, nil
}
