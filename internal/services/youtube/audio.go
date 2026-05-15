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

func useYTDLP(youtubeId string) (string, error) {
	bin, err := Ensure()
	if err != nil {
		return "", err
	}

	cookieStatus := CookieStatus()

	playerClients := []string{
		"web",
		"android",
		"ios",
	}

	var attempts []string

	for _, client := range playerClients {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

		args := append(cookiesArgs(),
			"--extractor-args", fmt.Sprintf("youtube:player_client=%s", client),
			"-f", "bestaudio[ext=m4a]",
			"-g", YOUTUBE_URL+youtubeId,
		)

		cmd := exec.CommandContext(ctx, bin, args...)
		out, err := cmd.CombinedOutput()
		cancel()

		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}

		attempts = append(attempts,
			fmt.Sprintf("client=%s: %s", client, strings.TrimSpace(string(out))),
		)
	}

	{
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		args := append(cookiesArgs(),
			"-f", "bestaudio",
			"-g", YOUTUBE_URL+youtubeId,
		)

		cmd := exec.CommandContext(ctx, bin, args...)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return strings.TrimSpace(string(out)), nil
		}

		attempts = append(attempts,
			fmt.Sprintf("fallback (any format): %s", strings.TrimSpace(string(out))),
		)
	}

	msg := fmt.Sprintf("yt-dlp failed (%s)", cookieStatus)

	if !HasCookies() {
		msg += "\nUpload your YouTube cookies via POST /api/user/cookies/ with a cookies.txt file exported from your logged-in browser (use a browser extension like 'Get cookies.txt')"
	} else {
		msg += "\nCookies file exists but YouTube rejected it. The cookies may be expired or invalid. Try re-exporting cookies.txt from your browser while logged into YouTube."
	}

	msg += "\n\nAttempts:\n" + strings.Join(attempts, "\n")

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
	return youtubeSourceBucket.SetString(id, youtubeId)
}