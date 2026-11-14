package youtube

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The player seeks with a byte range. The proxy must ask googlevideo for that
// same range and hand back its answer untouched: a body that starts where the
// headers do not say is a seek into the wrong bytes, which the player reports
// as the end of the track.
func TestRelayForwardsRange(t *testing.T) {
	body := strings.Repeat("0123456789", 100)
	var gotRange string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		start := 400
		w.Header().Set("Content-Type", "audio/mp4")
		w.Header().Set("Content-Range", "bytes 400-999/1000")
		w.Header().Set("Content-Length", "600")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, body[start:])
	}))
	defer upstream.Close()

	proxyMu.Lock()
	proxyStreams["seek"] = stream{url: upstream.URL, mime: "audio/mp4"}
	proxyMu.Unlock()
	defer func() {
		proxyMu.Lock()
		delete(proxyStreams, "seek")
		proxyMu.Unlock()
	}()

	req := httptest.NewRequest(http.MethodGet, "/stream/seek", nil)
	req.Header.Set("Range", "bytes=400-")
	rec := httptest.NewRecorder()
	relay(rec, req)

	if gotRange != "bytes=400-999" {
		t.Fatalf("upstream got range %q, want bytes=400-999", gotRange)
	}
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206", rec.Code)
	}
	if cr := rec.Header().Get("Content-Range"); cr != "bytes 400-999/1000" {
		t.Fatalf("content-range = %q", cr)
	}
	if got := rec.Body.String(); got != body[400:] {
		t.Fatalf("body carried %d bytes, want the 600 from offset 400", len(got))
	}
}

// A player request without a range is answered as googlevideo answers it: no
// invented 206, and the format's own content type when googlevideo omits it.
func TestRelayWithoutRange(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header()["Content-Type"] = nil // no type at all, as googlevideo can answer
		w.Header().Set("Content-Length", "3")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "abc")
	}))
	defer upstream.Close()

	proxyMu.Lock()
	proxyStreams["whole"] = stream{url: upstream.URL, mime: "audio/webm"}
	proxyMu.Unlock()
	defer func() {
		proxyMu.Lock()
		delete(proxyStreams, "whole")
		proxyMu.Unlock()
	}()

	rec := httptest.NewRecorder()
	relay(rec, httptest.NewRequest(http.MethodGet, "/stream/whole", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "audio/webm" {
		t.Fatalf("content-type = %q, want audio/webm", ct)
	}
	if rec.Header().Get("Content-Range") != "" {
		t.Fatalf("content-range set on a whole answer")
	}
}

func TestUnknownStreamIs404(t *testing.T) {
	rec := httptest.NewRecorder()
	relay(rec, httptest.NewRequest(http.MethodGet, "/stream/missing", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestUnpinnedKeepsSignature(t *testing.T) {
	raw := "https://rr1.googlevideo.com/videoplayback?expire=1&sig=abc%3D&range=0-1234&n=x%2By&mime=audio%2Fwebm"
	got := unpinned(raw)
	if strings.Contains(got, "range=") {
		t.Fatalf("range param survived: %s", got)
	}
	for _, want := range []string{"expire=1", "sig=abc%3D", "n=x%2By", "mime=audio%2Fwebm"} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q lost from %s", want, got)
		}
	}
}

func TestStreamMime(t *testing.T) {
	cases := map[string]string{
		"https://x/videoplayback?mime=audio%2Fmp4%3B+codecs%3D%22mp4a.40.2%22": "audio/mp4",
		"https://x/videoplayback?mime=audio%2Fwebm%3B+codecs%3D%22opus%22":     "audio/webm",
		"https://x/videoplayback?mime=video%2Fmp4":                             "audio/mp4",
		"https://x/videoplayback":                                              "audio/mp4",
	}
	for raw, want := range cases {
		if got := streamMime(raw); got != want {
			t.Errorf("streamMime(%q) = %q, want %q", raw, got, want)
		}
	}
}
