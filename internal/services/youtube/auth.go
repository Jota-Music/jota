package youtube

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

const (
	authKey    = "cookies"
	accountKey = "account"
)

var (
	authMu     sync.RWMutex
	authCookie string
	authLoaded bool
)

// Cookies returns the currently stored account cookie string (normalized), so it
// can be exported for use on another device.
func Cookies() string { return cookies() }

// cookies returns the account cookie string, preferring YOUTUBE_COOKIES so a
// headless setup can be configured without the UI.
func cookies() string {
	if env := os.Getenv("YOUTUBE_COOKIES"); env != "" {
		// A headless setup may export a cookies.txt, which still has to become a
		// Cookie header.
		return normalizeCookies(env)
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
	if sidValue(normalized) == "" {
		return errors.New("cookies missing SAPISID")
	}
	if err := authBucket.SetString(authKey, normalized); err != nil {
		return err
	}

	authMu.Lock()
	authCookie = normalized
	authLoaded = true
	authMu.Unlock()
	// The name belongs to the cookies that answered it.
	_ = authBucket.Delete(accountKey)
	// The cached list belongs to the account that was signed in before.
	_ = playlistBucket.Delete(myPlaylistsKey)
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
		// A jar keeps both copies of SID, SAPISID and friends: the .google.com
		// one and the .youtube.com one. A browser sends only the youtube copy to
		// youtube.com, and sending both makes YouTube read the request as
		// anonymous ("The playlist does not exist"), so keep one per name.
		picked := make(map[string]cookieValueRank)
		var order []string
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
			rank := cookieDomainRank(domain)
			if name == "" || rank == 0 {
				continue
			}
			if old, ok := picked[name]; !ok {
				order = append(order, name)
			} else if rank <= old.rank {
				continue
			}
			picked[name] = cookieValueRank{value: value, rank: rank}
		}
		pairs := make([]string, 0, len(order))
		for _, name := range order {
			pairs = append(pairs, name+"="+picked[name].value)
		}
		return strings.Join(pairs, "; ")
	}

	raw = strings.TrimPrefix(strings.TrimSpace(raw), "Cookie:")
	parts := strings.Split(raw, ";")
	pairs := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, _, _ := strings.Cut(part, "=")
		if seen[name] {
			continue
		}
		seen[name] = true
		pairs = append(pairs, part)
	}
	return strings.Join(pairs, "; ")
}

// cookieValueRank is one candidate value for a cookie name plus how specific its
// domain is to a youtube.com request.
type cookieValueRank struct {
	value string
	rank  int
}

// cookieDomainRank orders a jar row by whether a youtube.com request would carry
// it. The .youtube.com copy wins over the .google.com one, and a name that only
// exists on google.com is still kept so nothing is lost.
func cookieDomainRank(domain string) int {
	switch {
	case strings.Contains(domain, "youtube.com"):
		return 2
	case strings.Contains(domain, "google.com"):
		return 1
	}
	return 0
}

func ClearCookies() error {
	authMu.Lock()
	authCookie = ""
	authLoaded = true
	authMu.Unlock()
	_ = authBucket.Delete(accountKey)
	return authBucket.Delete(authKey)
}

func HasAuth() bool {
	return sidValue(cookies()) != ""
}

// sessionValue returns the cookie that marks a full signed-in session. The
// sign-in window cannot read it: Google marks __Secure-3PSID and SSID HttpOnly,
// so document.cookie never carries them and the page can only hand over the
// signing cookies. Its presence is what tells a real export from that partial
// set.
func sessionValue(raw string) string {
	for _, name := range []string{"__Secure-3PSID", "__Secure-1PSID", "SSID"} {
		if v := cookieValue(raw, name); v != "" {
			return v
		}
	}
	return ""
}

// sidValue returns the signing secret, in preference order. SAPISID is often
// absent in modern exports; __Secure-3PAPISID covers it.
func sidValue(raw string) string {
	for _, name := range []string{"SAPISID", "__Secure-3PAPISID", "__Secure-1PAPISID"} {
		if v := cookieValue(raw, name); v != "" {
			return v
		}
	}
	return ""
}

func cookieValue(raw, name string) string {
	for part := range strings.SplitSeq(raw, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && k == name {
			return v
		}
	}
	return ""
}

// Full reports whether a raw cookie string carries the private session ones.
// A full session can open the account's own lists; a partial one only signs
// requests.
func Full(raw string) bool {
	return sessionValue(raw) != ""
}

// SessionFull reports whether the stored cookies carry the private session ones
// (__Secure-3PSID, __Secure-1PSID or SSID). A set without them signs requests
// but stays anonymous to YouTube, so the account's own lists and any private
// playlist stay closed whatever the signature says.
func SessionFull() bool {
	return sessionValue(cookies()) != ""
}

// Verify asks YouTube whether the stored cookies are a signed-in session. A
// partial export still signs requests, so nothing looks wrong until a private
// resource is asked for; this turns that silence into an answer the user can
// act on.
func Verify() error {
	cookie := cookies()
	if sidValue(cookie) == "" {
		return errors.New("cookies missing SAPISID")
	}

	req, err := http.NewRequest("GET", "https://www.youtube.com/", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", webUserAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	for k, v := range authHeaders("https://www.youtube.com") {
		req.Header.Set(k, v)
	}

	resp, err := visitorClient.Do(req)
	if err != nil {
		return fmt.Errorf("verifying the YouTube session: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("verifying the YouTube session: %w", err)
	}
	// Only an answer from YouTube itself counts. A bot check or a transient
	// failure says nothing about the cookies, and dropping a session that works
	// would be worse than keeping one that does not.
	switch {
	case bytes.Contains(body, []byte(`"LOGGED_IN":true`)):
		return nil
	case bytes.Contains(body, []byte(`"LOGGED_IN":false`)):
		return errSession
	default:
		return nil
	}
}

// Account returns the account the stored cookies belong to, the way Spotify's
// panel names the user it is signed in as. A session the sign-in window handed
// over cannot answer: YouTube does not recognise it. The answer is remembered
// until the cookies change, because it costs a request and never moves.
func Account() string {
	cookie := cookies()
	if sessionValue(cookie) == "" {
		return ""
	}
	if cached, err := authBucket.GetString(accountKey); err == nil && cached != "" {
		return cached
	}
	name, err := accountName(cookie)
	if err != nil || name == "" {
		return ""
	}
	_ = authBucket.SetString(accountKey, name)
	return name
}

// accountName asks YouTube who a jar belongs to, reading the profile the guide
// carries.
func accountName(raw string) (string, error) {
	if sidValue(raw) == "" {
		return "", nil
	}
	payload := map[string]any{"context": map[string]any{"client": clientContext(webClient())}}
	data, err := retryAs(raw, webClient(), guideEndpoint, payload, 2)
	if err != nil {
		return "", err
	}

	var res struct {
		UserProfile struct {
			AccountName string `json:"accountName"`
			Email       string `json:"email"`
		} `json:"userProfile"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return "", err
	}
	switch {
	case res.UserProfile.AccountName != "" && res.UserProfile.Email != "":
		return res.UserProfile.AccountName + " <" + res.UserProfile.Email + ">", nil
	case res.UserProfile.Email != "":
		return res.UserProfile.Email, nil
	}
	return res.UserProfile.AccountName, nil
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
	return authHeadersFor(cookies(), origin)
}

// authHeadersFor signs a request with one cookie jar, which may be a browser's
// jar rather than the stored session.
func authHeadersFor(cookie, origin string) map[string]string {
	sapisid := sidValue(cookie)
	if sapisid == "" {
		return nil
	}

	// YouTube validates each scheme against its own cookie, so a set that only
	// carries __Secure-3PAPISID cannot be signed as SAPISIDHASH. Sending every
	// scheme it can, the way yt-dlp does, keeps such a set usable.
	ts := time.Now().Unix()
	sign := func(scheme, secret string) string {
		sum := sha1.Sum(fmt.Appendf(nil, "%d %s %s", ts, secret, origin))
		return fmt.Sprintf("%s %d_%s", scheme, ts, hex.EncodeToString(sum[:]))
	}
	auth := sign("SAPISIDHASH", sapisid)
	if v := cookieValue(cookie, "__Secure-1PAPISID"); v != "" {
		auth += " " + sign("SAPISID1PHASH", v)
	}
	if v := cookieValue(cookie, "__Secure-3PAPISID"); v != "" {
		auth += " " + sign("SAPISID3PHASH", v)
	}

	return map[string]string{
		"Authorization":   auth,
		"X-Origin":        origin,
		"X-Goog-AuthUser": "0",
		"Cookie":          cookie,
	}
}
