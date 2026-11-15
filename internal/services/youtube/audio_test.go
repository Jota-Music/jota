package youtube

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/music"
)

func resetPotCache(t *testing.T) {
	t.Helper()
	potMu.Lock()
	orig := potCache
	potCache = map[string]potEntry{}
	potMu.Unlock()
	t.Cleanup(func() {
		potMu.Lock()
		potCache = orig
		potMu.Unlock()
	})
}

// resetSticky clears the remembered client so a test's order does not depend on
// what an earlier test resolved, restoring the previous value afterwards.
func resetSticky(t *testing.T) {
	t.Helper()
	orig := youtubeCache.client()
	youtubeCache.forgetClient()
	t.Cleanup(func() {
		if orig != "" {
			youtubeCache.rememberClient(orig)
			return
		}
		youtubeCache.forgetClient()
	})
}

func TestClientOrderPutsStickyClientFirst(t *testing.T) {
	resetSticky(t)
	youtubeCache.rememberClient("IOS")

	want := []string{"IOS"}
	for _, c := range allClients() {
		if c.Name != "IOS" {
			want = append(want, c.Name)
		}
	}

	order := clientOrder()
	if len(order) != len(want) {
		t.Fatalf("clients = %d, want %d", len(order), len(want))
	}
	for i, name := range want {
		if order[i].Name != name {
			t.Fatalf("order[%d] = %q, want %q", i, order[i].Name, name)
		}
	}
}

func TestWebPoTokenProviderFailures(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"server error", http.StatusInternalServerError, `{}`},
		{"bad json", http.StatusOK, `not-json`},
		{"empty token", http.StatusOK, `{"poToken":""}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.status)
				_, _ = io.WriteString(w, c.body)
			}))
			defer server.Close()
			t.Setenv("YOUTUBE_POTOKEN_PROVIDER", server.URL)
			resetPotCache(t)

			if got := webPoToken("vid-fail"); got != "" {
				t.Fatalf("token = %q, want empty", got)
			}
		})
	}

	t.Run("unreachable", func(t *testing.T) {
		t.Setenv("YOUTUBE_POTOKEN_PROVIDER", "http://127.0.0.1:1")
		resetPotCache(t)

		if got := webPoToken("vid-unreachable"); got != "" {
			t.Fatalf("token = %q, want empty", got)
		}
	})
}

// TestWebClientWithProviderIntegration is the automated validation gate for the
// blocking issue: with a PO token provider running, the WEB client must produce
// a stream. It is skipped unless YOUTUBE_POTOKEN_PROVIDER is set.
func TestWebClientWithProviderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: reaches YouTube")
	}
	if poTokenProviderURL() == "" {
		t.Skip("set YOUTUBE_POTOKEN_PROVIDER to run the blocking gate")
	}

	if token := webPoToken("E9s9BNZFQLA"); token == "" {
		t.Fatal("provider returned no PO token")
	}

	raw, err := playerStreamURL(webClient(), "E9s9BNZFQLA")
	if err != nil {
		t.Fatalf("WEB client failed with a provider token: %v", err)
	}
	if raw == "" {
		t.Fatal("empty stream URL")
	}
}

func TestPlayerPayloadAttachesPoTokenForWeb(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/get_pot" {
			t.Errorf("provider path = %q, want /get_pot", r.URL.Path)
		}
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		expires := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
		fmt.Fprintf(w, `{"poToken":"tok-123","expiresAt":%q}`, expires)
	}))
	defer server.Close()

	t.Setenv("YOUTUBE_POTOKEN_PROVIDER", server.URL)
	resetPotCache(t)
	seedVisitor(t, "v-token")

	payload := playerPayload(webClient(), "vid12345678")
	client := payload["context"].(map[string]any)["client"].(map[string]any)
	if client["visitorData"] != "v-token" {
		t.Fatalf("web client visitorData = %v, want v-token", client["visitorData"])
	}
	sid, ok := payload["serviceIntegrityDimensions"].(map[string]any)
	if !ok || sid["poToken"] != "tok-123" {
		t.Fatalf("poToken not attached: %#v", payload["serviceIntegrityDimensions"])
	}

	playerPayload(webClient(), "vid12345678")
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("provider hit %d times, want 1 (cached by video id)", got)
	}

	if _, ok := playerPayload(preferredClient, "vid12345678")["serviceIntegrityDimensions"]; ok {
		t.Fatal("non-web client must not carry a poToken")
	}
}

func TestPlayerPayloadWithoutProviderHasNoPoToken(t *testing.T) {
	t.Setenv("YOUTUBE_POTOKEN_PROVIDER", "")

	if _, ok := playerPayload(webClient(), "vid12345678")["serviceIntegrityDimensions"]; ok {
		t.Fatal("no provider configured, so no poToken expected")
	}
}

func TestBestAudioPrefersMP4(t *testing.T) {
	cases := []struct {
		name    string
		formats []format
		want    int
		ok      bool
	}{
		{
			name: "prefers mp4 over higher webm itag",
			formats: []format{
				{Itag: 251, MimeType: `audio/webm; codecs="opus"`, URL: "webm"},
				{Itag: 140, MimeType: `audio/mp4; codecs="mp4a.40.2"`, URL: "mp4"},
			},
			want: 140,
			ok:   true,
		},
		{
			name:    "falls back to webm when no mp4",
			formats: []format{{Itag: 251, MimeType: `audio/webm; codecs="opus"`, URL: "webm"}},
			want:    251,
			ok:      true,
		},
		{
			name: "highest mp4 wins within tier",
			formats: []format{
				{Itag: 139, MimeType: "audio/mp4", URL: "a"},
				{Itag: 140, MimeType: "audio/mp4", URL: "b"},
			},
			want: 140,
			ok:   true,
		},
		{
			name: "ignores non-audio and empty url",
			formats: []format{
				{Itag: 137, MimeType: "video/mp4", URL: "v"},
				{Itag: 140, MimeType: "audio/mp4", URL: ""},
			},
			ok: false,
		},
		{name: "empty", formats: nil, ok: false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := bestAudio(c.formats)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v", ok, c.ok)
			}
			if ok && got.Itag != c.want {
				t.Fatalf("itag = %d, want %d", got.Itag, c.want)
			}
		})
	}
}

func TestBestAudioPrefersWebMWhereMP4CannotSeek(t *testing.T) {
	preferWebM = true
	t.Cleanup(func() { preferWebM = false })

	formats := []format{
		{Itag: 140, MimeType: `audio/mp4; codecs="mp4a.40.2"`, URL: "mp4"},
		{Itag: 251, MimeType: `audio/webm; codecs="opus"`, URL: "webm"},
	}
	got, ok := bestAudio(formats)
	if !ok || got.Itag != 251 {
		t.Fatalf("itag = %d (ok %v), want 251", got.Itag, ok)
	}

	if got, _ := bestAudio([]format{{Itag: 140, MimeType: "audio/mp4", URL: "mp4"}}); got.Itag != 140 {
		t.Fatalf("itag = %d, want the MP4 fallback when no WebM exists", got.Itag)
	}
}

func TestBestAudioPrefersOriginalTrackOverDub(t *testing.T) {
	original := &audioTrack{ID: "es-US.4", Default: true}
	dub := &audioTrack{ID: "ar.10"}

	// The order IOS answers QUVWPQOSsio with: the Arabic dub comes first and
	// shares every itag with the original, so the itag tie picks it.
	formats := []format{
		{Itag: 139, MimeType: "audio/mp4", URL: "dub-low", AudioTrack: dub},
		{Itag: 139, MimeType: "audio/mp4", URL: "original-low", AudioTrack: original},
		{Itag: 140, MimeType: "audio/mp4", URL: "dub", AudioTrack: dub},
		{Itag: 140, MimeType: "audio/mp4", URL: "original", AudioTrack: original},
	}
	got, ok := bestAudio(formats)
	if !ok || got.URL != "original" {
		t.Fatalf("url = %q (ok %v), want the original track", got.URL, ok)
	}

	if got, _ := bestAudio([]format{{Itag: 140, MimeType: "audio/mp4", URL: "only-dub", AudioTrack: dub}}); got.URL != "only-dub" {
		t.Fatalf("url = %q, want the dub when it is the only track", got.URL)
	}
}

func TestDubbed(t *testing.T) {
	dub := "https://rr1---sn-x.googlevideo.com/videoplayback?itag=140&xtags=acont%3Ddubbed-auto%3Alang%3Dar&n=abc"
	original := "https://rr1---sn-x.googlevideo.com/videoplayback?itag=140&xtags=acont%3Doriginal%3Alang%3Des-US&n=abc"
	plain := "https://rr1---sn-x.googlevideo.com/videoplayback?itag=140&n=abc"

	if !dubbed(dub) {
		t.Fatal("a dubbed-auto URL was not recognised")
	}
	if dubbed(original) || dubbed(plain) {
		t.Fatal("an original track was reported as dubbed")
	}
}

func TestStreamPlayable(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Fatalf("expected HEAD, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	forbiddenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer forbiddenServer.Close()

	notFoundServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundServer.Close()

	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"ok returns true", okServer.URL, true},
		{"403 returns false", forbiddenServer.URL, false},
		{"404 returns false", notFoundServer.URL, false},
		{"unreachable returns false", "http://127.0.0.1:1/", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := streamPlayable(preferredClient, c.url)
			if got != c.want {
				t.Fatalf("streamPlayable(%q) = %v, want %v", c.url, got, c.want)
			}
		})
	}
}

func TestAudioCacheStillValid(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer badServer.Close()

	future := time.Now().Add(time.Hour).Unix()

	cases := []struct {
		name string
		a    *music.Audio
		want bool
	}{
		{"nil audio", nil, false},
		{"empty url", &music.Audio{Url: "", ExpireAt: future}, false},
		{"expired", &music.Audio{Url: okServer.URL, ExpireAt: time.Now().Add(-time.Minute).Unix()}, false},
		{"valid with stored client", &music.Audio{Url: okServer.URL, ExpireAt: future, ClientName: "ANDROID_VR"}, true},
		{"invalid with stored client", &music.Audio{Url: badServer.URL, ExpireAt: future, ClientName: "ANDROID_VR"}, false},
		{"old cache without client name falls back to preferred", &music.Audio{Url: okServer.URL, ExpireAt: future}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := audioCacheStillValid(c.a)
			if got != c.want {
				t.Fatalf("audioCacheStillValid() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestClientByName(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   string
		wantOk bool
	}{
		{"empty defaults to preferred", "", "VISIONOS", true},
		{"preferred", "VISIONOS", "VISIONOS", true},
		{"android vr", "ANDROID_VR", "ANDROID_VR", true},
		{"ios", "IOS", "IOS", true},
		{"web", "WEB", "WEB", true},
		{"unknown", "WATCH", "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := clientByName(c.input)
			if ok != c.wantOk {
				t.Fatalf("clientByName(%q) ok = %v, want %v", c.input, ok, c.wantOk)
			}
			if got.Name != c.want {
				t.Fatalf("clientByName(%q) name = %q, want %q", c.input, got.Name, c.want)
			}
		})
	}
}

func TestAudioURLRejectsUnplayableStreams(t *testing.T) {
	resetSticky(t)
	oldFn := playerStreamURLFn
	defer func() { playerStreamURLFn = oldFn }()

	playerStreamURLFn = func(c clientConfig, videoID string) (string, error) {
		return "http://127.0.0.1:1/unplayable", nil
	}

	_, _, err := audioURL("dQw4w9WgXcQ")
	if err == nil {
		t.Fatal("expected error for unplayable streams, got nil")
	}
}

func TestAudioURLDoesNotRefreshWhenFallbackWorks(t *testing.T) {
	resetSticky(t)
	oldFn := playerStreamURLFn
	defer func() { playerStreamURLFn = oldFn }()

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	visitorDataMu.Lock()
	origVisitor, origLast, origBlocked := visitorData, lastFetch, blockedUntil
	visitorData, lastFetch = "seed-token", time.Now()
	blockedUntil = time.Time{}
	visitorDataMu.Unlock()
	defer func() {
		visitorDataMu.Lock()
		visitorData, lastFetch, blockedUntil = origVisitor, origLast, origBlocked
		visitorDataMu.Unlock()
	}()

	playerStreamURLFn = func(c clientConfig, videoID string) (string, error) {
		if c.Name == preferredClient.Name {
			return "", ErrLoginRequired
		}
		return okServer.URL, nil
	}

	url, client, err := audioURL("dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("expected a fallback to play, got %v", err)
	}
	if url != okServer.URL || client.Name == preferredClient.Name {
		t.Fatalf("expected a non-preferred client, got %q/%q", url, client.Name)
	}

	visitorDataMu.RLock()
	got := visitorData
	visitorDataMu.RUnlock()
	if got != "seed-token" {
		t.Fatalf("visitor refreshed although a fallback worked: %q", got)
	}
}

func TestAudioURLRefreshesVisitorWhenNoClientPlays(t *testing.T) {
	resetSticky(t)
	oldFn := playerStreamURLFn
	defer func() { playerStreamURLFn = oldFn }()

	visitorDataMu.Lock()
	origVisitor, origLast, origBlocked := visitorData, lastFetch, blockedUntil
	visitorData, lastFetch = "seed-token", time.Now()
	blockedUntil = time.Time{}
	visitorDataMu.Unlock()
	defer func() {
		visitorDataMu.Lock()
		visitorData, lastFetch, blockedUntil = origVisitor, origLast, origBlocked
		visitorDataMu.Unlock()
	}()

	calls := 0
	playerStreamURLFn = func(c clientConfig, videoID string) (string, error) {
		calls++
		return "", ErrLoginRequired
	}

	_, _, err := audioURL("dQw4w9WgXcQ")
	if err == nil {
		t.Fatal("expected an error when every client is bot-checked")
	}
	if want := 2 * len(allClients()); calls != want {
		t.Fatalf("clients probed %d times, want %d (one refresh, one retry)", calls, want)
	}

	visitorDataMu.RLock()
	got := visitorData
	visitorDataMu.RUnlock()
	if got != "" {
		t.Fatalf("expected the visitor to be invalidated, got %q", got)
	}
}

func TestAudioURLReturnsFirstPlayableClient(t *testing.T) {
	resetSticky(t)
	oldFn := playerStreamURLFn
	defer func() { playerStreamURLFn = oldFn }()

	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	callCount := 0
	playerStreamURLFn = func(c clientConfig, videoID string) (string, error) {
		callCount++
		if c.Name == "ANDROID_VR" {
			return okServer.URL, nil
		}
		return "http://127.0.0.1:1/unplayable", nil
	}

	url, client, err := audioURL("dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if url != okServer.URL {
		t.Fatalf("url = %q, want %q", url, okServer.URL)
	}
	if client.Name != "ANDROID_VR" {
		t.Fatalf("client = %q, want ANDROID_VR", client.Name)
	}
	if callCount < 2 {
		t.Fatalf("expected fallback probes, got %d calls", callCount)
	}
}

// TestPlayerRequestSignsInAfterLoginRequired pins the fallback: the anonymous
// player answer is retried signed, and that second request is the one carrying
// the account cookies.
func TestPlayerRequestSignsInAfterLoginRequired(t *testing.T) {
	resetAuth(t)
	seedVisitor(t, "v-test")
	if err := SetCookies("SID=abc; SAPISID=sapisid123"); err != nil {
		t.Fatalf("seed cookies: %v", err)
	}

	var hits, signed int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if r.Header.Get("Cookie") == "" {
			_, _ = io.WriteString(w, `{"playabilityStatus":{"status":"LOGIN_REQUIRED"}}`)
			return
		}
		atomic.AddInt32(&signed, 1)
		_, _ = io.WriteString(w, `{"playabilityStatus":{"status":"OK"}}`)
	}))
	defer server.Close()

	data, err := playerRequest(preferredClient, server.URL, map[string]any{"videoId": "vid12345678"})
	if err != nil {
		t.Fatalf("playerRequest: %v", err)
	}
	if loginRequired(data) {
		t.Fatalf("expected the signed answer, got %s", data)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("requests = %d, want 2 (anonymous, then signed)", got)
	}
	if got := atomic.LoadInt32(&signed); got != 1 {
		t.Fatalf("signed requests = %d, want 1", got)
	}
}

func TestPlayerRequestSignsInForMembersOnly(t *testing.T) {
	resetAuth(t)
	seedVisitor(t, "v-test")
	if err := SetCookies("SID=abc; SAPISID=sapisid123"); err != nil {
		t.Fatalf("seed cookies: %v", err)
	}

	var signed int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") == "" {
			_, _ = io.WriteString(w, `{"playabilityStatus":{"status":"UNPLAYABLE","reason":"Join this channel to get access to members-only content like this video."}}`)
			return
		}
		atomic.AddInt32(&signed, 1)
		_, _ = io.WriteString(w, `{"playabilityStatus":{"status":"OK"}}`)
	}))
	defer server.Close()

	data, err := playerRequest(webClient(), server.URL, map[string]any{"videoId": "vid12345678"})
	if err != nil {
		t.Fatalf("playerRequest: %v", err)
	}
	if status, _ := playability(data); status != "OK" {
		t.Fatalf("expected the signed answer, got %s", data)
	}
	if got := atomic.LoadInt32(&signed); got != 1 {
		t.Fatalf("signed requests = %d, want 1", got)
	}
}

func TestPlayerRequestLeavesDeviceClientsAnonymousForMembersOnly(t *testing.T) {
	resetAuth(t)
	seedVisitor(t, "v-test")
	if err := SetCookies("SID=abc; SAPISID=sapisid123"); err != nil {
		t.Fatalf("seed cookies: %v", err)
	}

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if cookie := r.Header.Get("Cookie"); cookie != "" {
			t.Errorf("device client carried cookies: %q", cookie)
		}
		_, _ = io.WriteString(w, `{"playabilityStatus":{"status":"UNPLAYABLE","reason":"Join this channel to get access to members-only content like this video."}}`)
	}))
	defer server.Close()

	_, err := playerRequest(preferredClient, server.URL, map[string]any{"videoId": "vid12345678"})
	if err != nil {
		t.Fatalf("playerRequest: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("requests = %d, want 1 (only the web client signs)", got)
	}
}

func TestNsigGated(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"https://x/y?n=abc", true},
		{"https://x/y?foo=1", false},
		{"https://x/y", false},
		{"not-a-url?n=1", true},
	}
	for _, c := range cases {
		if got := nsigGated(c.url); got != c.want {
			t.Errorf("nsigGated(%q)=%v want %v", c.url, got, c.want)
		}
	}
}

func TestAudioURLSurfacesMembersOnly(t *testing.T) {
	resetSticky(t)
	oldFn := playerStreamURLFn
	defer func() { playerStreamURLFn = oldFn }()

	playerStreamURLFn = func(c clientConfig, videoID string) (string, error) {
		return "", ErrMembersOnly
	}

	_, _, err := audioURL("dQw4w9WgXcQ")
	if !errors.Is(err, ErrMembersOnly) {
		t.Fatalf("expected ErrMembersOnly, got %v", err)
	}
}

func TestAudioURLSurfacesNChallenge(t *testing.T) {
	resetSticky(t)
	oldFn := playerStreamURLFn
	defer func() { playerStreamURLFn = oldFn }()

	playerStreamURLFn = func(c clientConfig, videoID string) (string, error) {
		return "", ErrNChallenge
	}

	_, _, err := audioURL("dQw4w9WgXcQ")
	if !errors.Is(err, ErrNChallenge) {
		t.Fatalf("expected ErrNChallenge, got %v", err)
	}
}
func TestPlayerRequestStaysAnonymousWithoutCookies(t *testing.T) {
	resetAuth(t)
	seedVisitor(t, "v-test")

	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if cookie := r.Header.Get("Cookie"); cookie != "" {
			t.Errorf("anonymous request carried cookies: %q", cookie)
		}
		_, _ = io.WriteString(w, `{"playabilityStatus":{"status":"LOGIN_REQUIRED"}}`)
	}))
	defer server.Close()

	data, err := playerRequest(preferredClient, server.URL, map[string]any{"videoId": "vid12345678"})
	if err != nil {
		t.Fatalf("playerRequest: %v", err)
	}
	if !loginRequired(data) {
		t.Fatalf("expected the anonymous answer, got %s", data)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("requests = %d, want 1 (nothing to sign in with)", got)
	}
}
