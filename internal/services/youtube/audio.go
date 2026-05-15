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

func isBotError(out string) bool {
	return strings.Contains(out, "Sign in to confirm") || strings.Contains(out, "bot")
}

func isPrivateError(out string) bool {
	return strings.Contains(out, "Private video") || strings.Contains(out, "private")
}

func isUnavailableError(out string) bool {
	return strings.Contains(out, "Video unavailable") || strings.Contains(out, "This video is not available")
}

func isAgeRestrictedError(out string) bool {
	return strings.Contains(out, "age") || strings.Contains(out, "Age") || strings.Contains(out, "confirm your age")
}

func isGeoBlockedError(out string) bool {
	return strings.Contains(out, "blocked") || strings.Contains(out, "not available in your country")
}

func formatYTDLPError(out string) string {
	lines := strings.Split(out, "\n")
	var clean []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "WARNING:") {
			continue
		}
		clean = append(clean, line)
	}

	errMsg := strings.Join(clean, "; ")

	switch {
	case isBotError(out):
		return fmt.Sprintf("YouTube is blocking the request. %s", botHelp())
	case isPrivateError(out):
		return "This video is private."
	case isUnavailableError(out):
		return "This video is unavailable."
	case isAgeRestrictedError(out):
		return "This video is age-restricted."
	case isGeoBlockedError(out):
		return "This video is not available in your region."
	default:
		if errMsg != "" {
			return errMsg
		}
		return "Failed to fetch audio from YouTube."
	}
}

func botHelp() string {
	if HasCookies() {
		return "Your cookies may be expired. Upload fresh cookies via POST /api/user/cookies."
	}
	return "Upload YouTube cookies via POST /api/user/cookies to authenticate. See https://github.com/yt-dlp/yt-dlp/wiki/Extractors#exporting-youtube-cookies"
}

func cookiesArgs() []string {
	args := []string{
		"--js-runtimes", "node",
		"--remote-components", "ejs:github",
		"--extractor-retries", "3",
		"--throttled-rate", "100K",
	}
	if HasCookies() {
		args = append(args, "--cookies", cookiesPath())
	}
	return args
}

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

	cmd := exec.Command(bin, append(cookiesArgs(), args...)...)
	return cmd.CombinedOutput()
}

func useYTDLP(youtubeId string) (string, error) {
	bin, err := Ensure()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	args := append(cookiesArgs(), "-f", "bestaudio[ext=m4a]", "-g", YOUTUBE_URL+youtubeId)

	cmd := exec.CommandContext(ctx, bin, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New(formatYTDLPError(string(out)))
	}

	return strings.TrimSpace(string(out)), nil
}

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

	durStr := q.Get("dur")
	if durStr == "" {
		return nil, false
	}

	durFloat, err := strconv.ParseFloat(durStr, 64)
	if err != nil {
		return nil, false
	}

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

func ttlFromExpire(expireAt int64) (time.Duration, bool) {
	now := time.Now().UTC().Unix()

	ttl := expireAt - now
	if ttl <= 0 {
		return 0, false
	}

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

func GetAudio(youtubeId string) (*Audio, error) {
	if strings.TrimSpace(youtubeId) == "" {
		return nil, errors.New("no video ID provided")
	}

	var cached Audio
	err := youtubeSourceBucket.GetObject(youtubeId, &cached)
	if err == nil && audioCacheStillValid(&cached) {
		return &cached, nil
	}
	if err != nil && !errors.Is(err, kv.KeyNotFoundError) {
		return nil, fmt.Errorf("cache error: %w", err)
	}

	streamURL, err := useYTDLP(youtubeId)
	if err != nil {
		return nil, err
	}

	info, ok := getExpireAndDurationFromURL(streamURL)
	if !ok {
		return nil, errors.New("bad stream response from YouTube")
	}

	ttl, ok := ttlFromExpire(info.ExpireAt)
	if !ok {
		return nil, errors.New("stream URL expired before it could be cached")
	}

	audio := Audio{
		Url:      streamURL,
		Duration: info.Duration,
		ExpireAt: info.ExpireAt,
	}

	if err := youtubeSourceBucket.SetObject(youtubeId, audio, ttl); err != nil {
		return nil, fmt.Errorf("cache write error: %w", err)
	}

	return &audio, nil
}

func SetYoutubeId(id string, youtubeId string) error {
	return youtubeSourceBucket.SetString(id, youtubeId)
}