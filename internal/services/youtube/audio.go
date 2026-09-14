package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"jota/server/internal/kv"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var audioBucket = kv.UseBucket("youtube-audio")

// bestAudio picks the audio format with the fastest, most compatible start.
// audio/mp4 (AAC) is preferred over audio/webm (Opus): WebKitGTK's GStreamer
// backend streams MP4/AAC instantly but can stall on googlevideo's WebM/Opus,
// so "highest itag" alone would pick the slow one on desktop Linux.
func bestAudio(formats []format) (format, bool) {
	var pick format
	bestTier := -1
	bestItag := -1
	for _, f := range formats {
		if !strings.HasPrefix(f.MimeType, "audio/") || f.URL == "" {
			continue
		}
		tier := 0
		if strings.HasPrefix(f.MimeType, "audio/mp4") {
			tier = 1
		}
		if tier > bestTier || (tier == bestTier && f.Itag > bestItag) {
			bestTier, bestItag, pick = tier, f.Itag, f
		}
	}
	return pick, bestTier >= 0
}

func getExpireAndDurationFromURL(raw string) (*expireAndDuration, bool) {
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

	return &expireAndDuration{
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

func GetAudioURL(videoID string) (string, error) {
	if len(videoID) != 11 {
		return "", errors.New("invalid video ID length")
	}

	payload := map[string]any{
		"videoId":        videoID,
		"context":        map[string]any{"client": clientContext()},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	data, err := retryRequest("https://www.youtube.com/youtubei/v1/player", payload, true, 3)
	if err != nil {
		return "", fmt.Errorf("player request failed: %w", err)
	}

	var pr playerResponse
	if err := json.Unmarshal(data, &pr); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	if pr.PlayabilityStatus.Status == "LOGIN_REQUIRED" {
		invalidateVisitor()
		data, err = retryRequest("https://www.youtube.com/youtubei/v1/player", payload, true, 3)
		if err != nil {
			return "", fmt.Errorf("player request failed: %w", err)
		}
		if err := json.Unmarshal(data, &pr); err != nil {
			return "", fmt.Errorf("invalid JSON: %w", err)
		}
	}

	if pr.PlayabilityStatus.Status != "OK" {
		return "", fmt.Errorf("unavailable: %s", pr.PlayabilityStatus.Reason)
	}

	formats := append(pr.StreamingData.Formats, pr.StreamingData.AdaptiveFormats...)
	f, ok := bestAudio(formats)
	if !ok {
		return "", errors.New("no valid audio format found")
	}

	return f.URL, nil
}

func GetAudio(youtubeId string) (*Audio, error) {
	if strings.TrimSpace(youtubeId) == "" {
		return nil, errors.New("no video ID provided")
	}

	var cached Audio
	err := audioBucket.GetObject(youtubeId, &cached)
	if err == nil && audioCacheStillValid(&cached) {
		return &cached, nil
	}
	if err != nil && !errors.Is(err, kv.KeyNotFoundError) {
		return nil, fmt.Errorf("cache error: %w", err)
	}

	streamURL, err := GetAudioURL(youtubeId)
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
		VideoID:  youtubeId,
	}

	if err := audioBucket.SetObject(youtubeId, audio, ttl); err != nil {
		return nil, fmt.Errorf("cache write error: %w", err)
	}

	return &audio, nil
}

func GetCachedAudioBySpotifyId(spotifyId string) *Audio {
	var cached Audio
	if err := audioBucket.GetObject("spotify:"+spotifyId, &cached); err == nil {
		if audioCacheStillValid(&cached) {
			return &cached
		}
	}
	return nil
}

func GetCachedAudioByYoutubeId(youtubeId string) *Audio {
	var cached Audio
	if err := audioBucket.GetObject(youtubeId, &cached); err == nil {
		if audioCacheStillValid(&cached) {
			return &cached
		}
	}
	return nil
}

func SaveAudioForSpotifyId(spotifyId string, audio Audio, ttl time.Duration) {
	_ = audioBucket.SetObject("spotify:"+spotifyId, audio, ttl)
}
