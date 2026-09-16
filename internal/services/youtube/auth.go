package youtube

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/kv"
)

// Signing in with a Google account is the reliable way past YouTube's bot
// check. The app stores the account cookie string and signs innertube requests
// with SAPISIDHASH, the same mechanism yt-dlp uses. Doing this can get the
// account banned by Google, so the UI must warn the user before storing it.
var authBucket = kv.UseBucket("youtube-auth")

const authKey = "cookies"

var (
	authMu     sync.RWMutex
	authCookie string
	authLoaded bool
)

// cookies returns the account cookie string, preferring YOUTUBE_COOKIES so a
// headless setup can be configured without the UI.
func cookies() string {
	if env := os.Getenv("YOUTUBE_COOKIES"); env != "" {
		return env
	}

	authMu.RLock()
	if authLoaded {
		c := authCookie
		authMu.RUnlock()
		return c
	}
	authMu.RUnlock()

	authMu.Lock()
	defer authMu.Unlock()
	authLoaded = true
	if stored, err := authBucket.GetString(authKey); err == nil {
		authCookie = stored
	}
	return authCookie
}

// SetCookies stores the account cookie string. It must carry SAPISID, which is
// what the request signature is derived from. Accepts either a raw Cookie header
// ("a=1; b=2") or a Netscape cookies.txt export.
func SetCookies(raw string) error {
	normalized := normalizeCookies(raw)
	if normalized == "" {
		return errors.New("empty cookies")
	}
	if cookieValue(normalized, "SAPISID") == "" {
		return errors.New("cookies missing SAPISID")
	}
	if err := authBucket.SetString(authKey, normalized); err != nil {
		return err
	}

	authMu.Lock()
	authCookie = normalized
	authLoaded = true
	authMu.Unlock()
	return nil
}

// normalizeCookies turns a raw Cookie header or a Netscape cookies.txt export
// into the header form, keeping only YouTube/Google cookies.
func normalizeCookies(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.Contains(raw, "\t") {
		var pairs []string
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimRight(strings.TrimPrefix(line, "#HttpOnly_"), "\r")
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Split(line, "\t")
			if len(fields) < 7 {
				continue
			}
			domain, name, value := fields[0], fields[5], fields[6]
			if name == "" || (!strings.Contains(domain, "youtube.com") && !strings.Contains(domain, "google.com")) {
				continue
			}
			pairs = append(pairs, name+"="+value)
		}
		return strings.Join(pairs, "; ")
	}

	raw = strings.TrimPrefix(strings.TrimSpace(raw), "Cookie:")
	parts := strings.Split(raw, ";")
	pairs := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			pairs = append(pairs, part)
		}
	}
	return strings.Join(pairs, "; ")
}

func ClearCookies() error {
	authMu.Lock()
	authCookie = ""
	authLoaded = true
	authMu.Unlock()
	return authBucket.Delete(authKey)
}

func HasAuth() bool {
	return cookieValue(cookies(), "SAPISID") != ""
}

func cookieValue(raw, name string) string {
	for _, part := range strings.Split(raw, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && k == name {
			return v
		}
	}
	return ""
}

// originOf derives the request origin (scheme://host) the signature is bound to.
func originOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return "https://www.youtube.com"
	}
	return u.Scheme + "://" + u.Host
}

// authHeaders signs a request for the given origin. Returns nil when not signed
// in, so callers can send the request unauthenticated.
func authHeaders(origin string) map[string]string {
	cookie := cookies()
	sapisid := cookieValue(cookie, "SAPISID")
	if sapisid == "" {
		return nil
	}

	ts := time.Now().Unix()
	sum := sha1.Sum([]byte(fmt.Sprintf("%d %s %s", ts, sapisid, origin)))
	return map[string]string{
		"Authorization":   fmt.Sprintf("SAPISIDHASH %d_%s", ts, hex.EncodeToString(sum[:])),
		"X-Origin":        origin,
		"X-Goog-AuthUser": "0",
		"Cookie":          cookie,
	}
}
