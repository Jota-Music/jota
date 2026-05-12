package youtube

import (
	"context"
	"errors"
	"fmt"
	"jota/server/internal/kv"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var youtubeSourceBucket = kv.UseBucket("youtube-source")

const YOUTUBE_URL = "https://www.youtube.com/watch?v="

type Audio struct {
	Url      string `json:"url"`
	Duration int    `json:"duration"`
	ExpireAt int64  `json:"expireAt"`
}

func Run(args ...string) ([]byte, error) {
	bin, err := Ensure()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(bin, args...)
	return cmd.CombinedOutput()
}

func useYTDLP(youtubeId string) (string, error) {
	// Try browser-based extraction first (avoids bot detection)
	if info, err := getStreamURLViaBrowser(youtubeId); err == nil && info.URL != "" {
		// Reconstruct URL with expire for compatibility with existing parsing
		// The browser returns the stream URL directly
		return info.URL, nil
	}

	// Fallback to yt-dlp
	bin, err := Ensure()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		bin,
		"--js-runtimes", "node",
		"-f", "bestaudio[ext=m4a]",
		"-g",
		YOUTUBE_URL+youtubeId,
	)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("yt-dlp: %s | %w", strings.TrimSpace(stderr.String()), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// ===========================
// URL PARSER
// ===========================

type ExpireAndDuration struct {
	ExpireAt int64
	Duration int
}

func getExpireAndDurationFromURL(raw string) (*ExpireAndDuration, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}

	q := u.Query()

	// duration
	durStr := q.Get("dur")
	if durStr == "" {
		return nil, false
	}

	durFloat, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		return nil, false
	}

	// expire (unix seconds)
	expireStr := q.Get("expire")
	if expireStr == "" {
		return nil, false
	}

	expireInt, err := strconv.ParseInt(expireStr, 10, 64)
	if err != nil {
		return nil, false
	}

	return &ExpireAndDuration{
		ExpireAt: expireInt,
		Duration: int(durFloat),
	}, true
}

// ===========================
// TTL CALC (SAFE)
// ===========================

func ttlFromExpire(expireAt int64) (time.Duration, bool) {
	now := time.Now().UTC().Unix()

	ttl := expireAt - now
	if ttl <= 0 {
		return 0, false
	}

	// safety margin (evita edge expiry)
	ttl -= 30
	if ttl <= 0 {
		return 0, false
	}

	return time.Duration(ttl) * time.Second, true
}

func audioCacheStillValid(a *Audio) bool {
	if a == nil || a.Url == "" || a.ExpireAt <= 0 {
		return false
	}
	now := time.Now().UTC().Unix()
	return a.ExpireAt > now+30
}

// ===========================
// MAIN
// ===========================

func GetAudio(youtubeId string) (*Audio, error) {
	if strings.TrimSpace(youtubeId) == "" {
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

	streamURL, err := useYTDLP(youtubeId)
	if err != nil {
		return nil, err
	}

	info, ok := getExpireAndDurationFromURL(streamURL)
	if !ok {
		return nil, errors.New("failed to parse stream url metadata")
	}

	ttl, ok := ttlFromExpire(info.ExpireAt)
	if !ok {
		return nil, errors.New("stream already expired")
	}

	audio := Audio{
		Url:      streamURL,
		Duration: info.Duration,
		ExpireAt: info.ExpireAt,
	}

	if err := youtubeSourceBucket.SetObject(youtubeId, audio, ttl); err != nil {
		return nil, fmt.Errorf("youtube audio cache write: %w", err)
	}

	return &audio, nil
}

func SetYoutubeId(id string, youtubeId string) error {
	// store as raw string to be compatible with GetString
	return youtubeSourceBucket.SetString(id, youtubeId)
}
