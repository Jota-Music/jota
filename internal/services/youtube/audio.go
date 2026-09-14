package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var audioBucket = kv.UseBucket("youtube-audio")

var probeClient = &http.Client{Timeout: 8 * time.Second}

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

func clientByName(name string) (clientConfig, bool) {
	if name == "" || name == preferredClient.Name {
		return preferredClient, true
	}
	for _, c := range fallbackClients {
		if c.Name == name {
			return c, true
		}
	}
	return clientConfig{}, false
}

// audioCacheStillValid reports whether a cached stream can still be used: not
// expired and actually answering the player's first request with the same
// client headers it was resolved with. A cached URL that googlevideo now 403s
// must be re-resolved, or the player retries it forever.
func audioCacheStillValid(a *music.Audio) bool {
	if a == nil || a.Url == "" || a.ExpireAt <= 0 {
		return false
	}
	now := time.Now().UTC().Unix()
	if a.ExpireAt <= now+30 {
		return false
	}
	c, _ := clientByName(a.ClientName)
	return streamPlayable(c, a.Url)
}

// playerStreamURLFn is a test hook; do not assign in production code.
var playerStreamURLFn = playerStreamURL

func playerStreamURL(c clientConfig, videoID string) (string, error) {
	payload := map[string]any{
		"videoId":        videoID,
		"context":        map[string]any{"client": clientContext(c)},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	data, err := retryRequest(c, "https://www.youtube.com/youtubei/v1/player", payload, true, 3)
	if err != nil {
		return "", fmt.Errorf("player request failed: %w", err)
	}

	var pr playerResponse
	if err := json.Unmarshal(data, &pr); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	if pr.PlayabilityStatus.Status == "LOGIN_REQUIRED" {
		invalidateVisitor()
		data, err = retryRequest(c, "https://www.youtube.com/youtubei/v1/player", payload, true, 3)
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

// streamPlayable reports whether the stream answers the plain/HEAD request that
// GStreamer's souphttpsrc issues first. Some clients return URLs that only
// answer bounded Range requests (403 otherwise), which stalls playback.
func streamPlayable(c clientConfig, raw string) bool {
	req, err := http.NewRequest(http.MethodHead, raw, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := probeClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return false
	default:
		return true
	}
}

// audioURL resolves a playable stream, trying the preferred innertube client
// first and falling back to others when its URL is not usable by the player.
// It only returns URLs that pass a HEAD probe with the same headers the player
// will use.
func audioURL(videoID string) (string, clientConfig, error) {
	if len(videoID) != 11 {
		return "", clientConfig{}, errors.New("invalid video ID length")
	}

	var firstErr error

	for _, c := range allClients() {
		raw, err := playerStreamURLFn(c, videoID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if streamPlayable(c, raw) {
			if c.Name != preferredClient.Name {
				log.Printf("youtube: %s fell back to client %s", videoID, c.Name)
			}
			return raw, c, nil
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("client %s returned unplayable URL", c.Name)
		}
	}

	if firstErr != nil {
		return "", clientConfig{}, firstErr
	}
	return "", clientConfig{}, errors.New("no playable stream found")
}

func fetchAudio(youtubeId string) (*music.Audio, error) {
	if strings.TrimSpace(youtubeId) == "" {
		return nil, errors.New("no video ID provided")
	}

	var cached music.Audio
	err := audioBucket.GetObject(youtubeId, &cached)
	if err == nil && audioCacheStillValid(&cached) {
		return &cached, nil
	}
	if err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
		return nil, fmt.Errorf("cache error: %w", err)
	}

	streamURL, client, err := audioURL(youtubeId)
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

	audio := music.Audio{
		Url:        streamURL,
		Duration:   info.Duration,
		ExpireAt:   info.ExpireAt,
		VideoID:    youtubeId,
		ClientName: client.Name,
	}

	if err := audioBucket.SetObject(youtubeId, audio, ttl); err != nil {
		return nil, fmt.Errorf("cache write error: %w", err)
	}

	return &audio, nil
}

func cachedAudioBySong(cacheKey string) *music.Audio {
	var cached music.Audio
	if err := audioBucket.GetObject("spotify:"+cacheKey, &cached); err == nil {
		if audioCacheStillValid(&cached) {
			return &cached
		}
	}
	return nil
}

func cachedAudioByYoutube(youtubeId string) *music.Audio {
	var cached music.Audio
	if err := audioBucket.GetObject(youtubeId, &cached); err == nil {
		if audioCacheStillValid(&cached) {
			return &cached
		}
	}
	return nil
}

func saveAudioBySong(cacheKey string, audio music.Audio) {
	_ = audioBucket.SetObject("spotify:"+cacheKey, audio)
}
