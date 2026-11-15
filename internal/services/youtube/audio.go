package youtube

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jota-Music/jota/internal/music"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// probeClient validates a stream URL without pulling the stream. It shares the
// relay's transport on purpose: the probe that proves a URL also opens the
// connection the first chunk is served from, so a track's start pays one TLS
// handshake and not two.
var probeClient = &http.Client{Timeout: 8 * time.Second, Transport: proxyClient.Transport}

// ErrLoginRequired means YouTube wants a signed-in session: either the bot
// check, or an age-restricted video. Neither resolves without account cookies.
var ErrLoginRequired = errors.New("youtube: login required")

// ErrMembersOnly means the video is for the channel's members. YouTube answers
// it as UNPLAYABLE, so it only opens with the account's cookies.
var ErrMembersOnly = errors.New("youtube: members-only video")

// ErrNChallenge means the only URL YouTube handed out needs its own player to
// translate the `n` query param. googlevideo answers 403 to the untranslated
// one, and only the signed web/tv clients return such URLs.
var ErrNChallenge = errors.New("youtube: stream needs the n challenge")

// bestAudio picks the audio format with the fastest, most compatible start.
// audio/mp4 (AAC) is preferred over audio/webm (Opus): WebKitGTK's GStreamer
// backend streams MP4/AAC instantly but can stall on googlevideo's WebM/Opus,
// so "highest itag" alone would pick the slow one on desktop Linux. The order
// flips where the platform player cannot seek the fragmented MP4 (see
// preferWebM), because a track that cannot be seeked is worse than a slow start.
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
		if preferWebM && strings.HasPrefix(f.MimeType, "audio/webm") {
			tier = 2
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

	var durFloat float64
	durStr := q.Get("dur")
	if durStr == "" {
		// Some very long videos omit "dur" from the signed URL. The player can
		// still use the element's own duration, so do not fail the whole resolve.
		durFloat = 0
	} else {
		var parseErr error
		durFloat, parseErr = strconv.ParseFloat(durStr, 64)
		if parseErr != nil {
			return nil, false
		}
	}

	expireStr := q.Get("expire")
	if expireStr == "" {
		return nil, false
	}

	expireInt, err := strconv.ParseInt(expireStr, 10, 64)
	if err != nil {
		return nil, false
	}

	// Round up so the reported duration never falls below the real track: the
	// player uses it as the authoritative end boundary.
	return &expireAndDuration{
		ExpireAt: expireInt,
		Duration: int(math.Ceil(durFloat)),
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
	if name == "" {
		return preferredClient, true
	}
	for _, c := range allClients() {
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

// poTokenProviderURL points at a PO Token provider (e.g.
// bgutil-ytdlp-pot-provider). Set YOUTUBE_POTOKEN_PROVIDER to its base URL; when
// unset, the WEB client simply runs without a token.
func poTokenProviderURL() string {
	return strings.TrimRight(os.Getenv("YOUTUBE_POTOKEN_PROVIDER"), "/")
}

type potEntry struct {
	token   string
	expires time.Time
}

var (
	potClient = &http.Client{Timeout: 10 * time.Second}
	potMu     sync.Mutex
	potCache  = map[string]potEntry{}
)

// webPoToken returns a player-context PO token bound to the video ID, cached
// until it expires. Empty means no provider (or no token), and the WEB request
// is sent without one.
func webPoToken(videoID string) string {
	base := poTokenProviderURL()
	if base == "" {
		return ""
	}

	potMu.Lock()
	if e, ok := potCache[videoID]; ok && time.Now().Before(e.expires) {
		potMu.Unlock()
		return e.token
	}
	potMu.Unlock()

	body, err := json.Marshal(map[string]any{"content_binding": videoID})
	if err != nil {
		return ""
	}
	req, err := http.NewRequest("POST", base+"/get_pot", bytes.NewReader(body))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := potClient.Do(req)
	if err != nil {
		log.Printf("youtube: pot provider request failed: %v", err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("youtube: pot provider status %d", resp.StatusCode)
		return ""
	}

	var out struct {
		PoToken   string `json:"poToken"`
		ExpiresAt string `json:"expiresAt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.PoToken == "" {
		log.Printf("youtube: pot provider returned no token: %v", err)
		return ""
	}

	expires := time.Now().Add(time.Hour)
	if t, err := time.Parse(time.RFC3339, out.ExpiresAt); err == nil {
		expires = t
	}

	potMu.Lock()
	// Expired tokens are never read again; sweep them so a long session over
	// many videos does not grow the map without bound.
	if len(potCache) >= 128 {
		now := time.Now()
		for id, e := range potCache {
			if now.After(e.expires) {
				delete(potCache, id)
			}
		}
	}
	potCache[videoID] = potEntry{token: out.PoToken, expires: expires}
	potMu.Unlock()
	return out.PoToken
}

// playerPayload builds the innertube player body. The WEB client carries the
// visitor id and, when a provider is configured, a PO token — matching how
// yt-dlp feeds its `web` client.
func playerPayload(c clientConfig, videoID string) map[string]any {
	client := clientContext(c)
	payload := map[string]any{
		"videoId":        videoID,
		"context":        map[string]any{"client": client},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	if c.Name == webClientName {
		if visitor, _, err := getVisitorData(); err == nil && visitor != "" {
			client["visitorData"] = visitor
		}
		if token := webPoToken(videoID); token != "" {
			payload["serviceIntegrityDimensions"] = map[string]any{"poToken": token}
		}
	}

	return payload
}

const playerEndpoint = "https://www.youtube.com/youtubei/v1/player"

func playability(data []byte) (status, reason string) {
	var out struct {
		PlayabilityStatus struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
		} `json:"playabilityStatus"`
	}
	if json.Unmarshal(data, &out) != nil {
		return "", ""
	}
	return out.PlayabilityStatus.Status, out.PlayabilityStatus.Reason
}

// nsigGated reports whether a stream URL carries the player's `n` challenge.
// Only the signed web/tv clients sign their URLs with it, and googlevideo
// refuses the untranslated value, so a URL that has one is not playable as it
// comes.
func nsigGated(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Query().Get("n") != ""
}
func loginRequired(data []byte) bool {
	status, _ := playability(data)
	return status == "LOGIN_REQUIRED"
}

// membersOnly reports whether a player answer refused the video because it is
// for the channel's members. YouTube answers that as UNPLAYABLE rather than
// LOGIN_REQUIRED, so the reason is the only thing telling the two apart.
func membersOnly(data []byte) bool {
	status, reason := playability(data)
	return status == "UNPLAYABLE" &&
		(strings.Contains(reason, "members-only") || strings.Contains(reason, "Join this channel"))
}

// needsSession reports whether signing the request is what could unlock the
// answer. A bot check or an age-restricted video is answered LOGIN_REQUIRED and
// any client can sign it. Members-only content is answered UNPLAYABLE and only
// the web client reads the account from the cookies: the device clients ignore
// them, so signing there would spend the account on the same refusal.
func needsSession(c clientConfig, data []byte) bool {
	if loginRequired(data) {
		return true
	}
	return c.Name == webClientName && membersOnly(data)
}

// playerRequest asks the player anonymously first. Only a failed request or an
// answer that a session could unlock is retried signed: signing is what opens
// age-restricted and members-only videos, but it also risks the account, so it
// is never the default.
func playerRequest(c clientConfig, url string, payload map[string]any) ([]byte, error) {
	data, err := retryAnonymous(c, url, payload, 3)
	if cookies() == "" || (err == nil && !needsSession(c, data)) {
		return data, err
	}
	signed, signedErr := retryRequest(c, url, payload, 3)
	if signedErr != nil {
		return data, err
	}
	return signed, nil
}

// playerStreamURLFn is a test hook; do not assign in production code.
var playerStreamURLFn = playerStreamURL

func playerStreamURL(c clientConfig, videoID string) (string, error) {
	payload := playerPayload(c, videoID)

	data, err := playerRequest(c, playerEndpoint, payload)
	if err != nil {
		return "", fmt.Errorf("player request failed: %w", err)
	}

	var pr playerResponse
	if err := json.Unmarshal(data, &pr); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	if pr.PlayabilityStatus.Status == "LOGIN_REQUIRED" {
		return "", ErrLoginRequired
	}

	if membersOnly(data) {
		return "", ErrMembersOnly
	}

	if pr.PlayabilityStatus.Status != "OK" {
		return "", fmt.Errorf("unavailable: %s", pr.PlayabilityStatus.Reason)
	}

	formats := append(pr.StreamingData.Formats, pr.StreamingData.AdaptiveFormats...)
	f, ok := bestAudio(formats)
	if !ok {
		return "", errors.New("no valid audio format found")
	}
	if nsigGated(f.URL) {
		return "", ErrNChallenge
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

// audioURL resolves a playable stream, trying the client that last worked
// first and the rest in allClients order when its URL is not usable by the
// player. It only returns URLs that pass a HEAD probe with the same headers the
// player will use, and it remembers the winner so the next track starts there.
//
// A bot-checked client may only need a fresh visitor token, but refreshing hits
// YouTube's homepage, so it is deferred until no client worked at all. That way
// a healthy fallback (e.g. IOS while ANDROID_VR is bot-checked) never triggers a
// homepage fetch.
func audioURL(videoID string) (string, clientConfig, error) {
	if len(videoID) != 11 {
		return "", clientConfig{}, errors.New("invalid video ID length")
	}

	order := clientOrder()
	sticky := order[0].Name
	var firstErr error
	var sentinelErr error
	refreshed := false

	for {
		loginRequired := false

		for _, c := range order {
			raw, err := playerStreamURLFn(c, videoID)
			if err != nil {
				if errors.Is(err, ErrLoginRequired) {
					loginRequired = true
				}
				if errors.Is(err, ErrNChallenge) || errors.Is(err, ErrMembersOnly) {
					if sentinelErr == nil {
						sentinelErr = err
					}
				} else if firstErr == nil {
					firstErr = err
				}
				continue
			}
			if streamPlayable(c, raw) {
				if c.Name != sticky && !nsigGated(raw) {
					log.Printf("youtube: %s switched to client %s", videoID, c.Name)
					youtubeCache.rememberClient(c.Name)
				}
				// The player seeks the fragmented MP4 badly where WebM is
				// preferred, so a video without Opus is worth knowing about: it
				// is the only way the seek can still land at the end of the track.
				if mime := streamMime(raw); preferWebM && mime != "audio/webm" {
					log.Printf("youtube: %s has no opus audio, playing %s (seek may fail)", videoID, mime)
				}
				return raw, c, nil
			}
			if firstErr == nil {
				firstErr = fmt.Errorf("client %s returned unplayable URL", c.Name)
			}
		}

		if refreshed || !loginRequired {
			break
		}
		log.Printf("youtube: %s no client played, refreshing visitor once", videoID)
		invalidateVisitor()
		refreshed = true
	}

	if sentinelErr != nil {
		return "", clientConfig{}, sentinelErr
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

	cached := cachedAudio(youtubeId)
	if cached != nil {
		return cached, nil
	}

	streamURL, client, err := audioURL(youtubeId)
	if err != nil {
		if !errors.Is(err, ErrNChallenge) && !errors.Is(err, ErrMembersOnly) {
			return nil, err
		}
		// The account opens the video and YouTube still hands out a URL that
		// cannot be played as it comes: yt-dlp is the one that solves the
		// player's challenge for it.
		cause := err
		streamURL, client, err = externalAudio(youtubeId)
		if err != nil {
			return nil, fmt.Errorf("%w (%v)", cause, err)
		}
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

	keepAudio(youtubeId, audio, ttl)

	// Return the proxy URL so WebKit gets a simple localhost URL that
	// buffers and seeks properly.
	audio.Url = ProxyURL(youtubeId, streamURL)

	return &audio, nil
}
