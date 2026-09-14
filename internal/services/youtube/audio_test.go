package youtube

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/music"
)

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

func TestAudioURLReturnsFirstPlayableClient(t *testing.T) {
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
