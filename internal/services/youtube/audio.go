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

func cookiesArgs() []string {
	args := []string{"--js-runtimes", "node"}
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

func extractYtError(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ERROR:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "ERROR:"))
		}
	}
	if len(lines) > 0 {
		return strings.TrimSpace(lines[len(lines)-1])
	}
	return output
}

const userAgent = "Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.6422.147 Mobile Safari/537.36"

func useYTDLP(youtubeId string) (string, error) {
	bin, err := Ensure()
	if err != nil {
		return "", err
	}

	strategies := []struct {
		client  string
		headers bool
		cookies bool
	}{
		{client: "android", headers: true},
		{client: "android", headers: true, cookies: true},
		{client: "ios", headers: true, cookies: true},
		{client: "web", cookies: true},
	}

	var lastErr string

	for _, s := range strategies {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		args := []string{
			"--no-update",
			"--user-agent", userAgent,
			"--extractor-args", fmt.Sprintf("youtube:player_client=%s", s.client),
			"-f", "bestaudio[ext=m4a]",
			"-g", YOUTUBE_URL+youtubeId,
		}

		if s.cookies && HasCookies() {
			args = append(args, "--cookies", cookiesPath())
		}

		if s.headers {
			args = append(args,
				"--add-header", "Accept-Language: en-US,en;q=0.9",
				"--add-header", "Origin: https://www.youtube.com",
			)
		}

		cmd := exec.CommandContext(ctx, bin, args...)
		out, err := cmd.CombinedOutput()
		cancel()

		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}

		lastErr = extractYtError(string(out))
	}

	msg := fmt.Sprintf("can't get audio: %s", lastErr)

	if !HasCookies() {
		msg += " (upload cookies via POST /api/user/cookies/)"
	}

	return "", errors.New(msg)
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