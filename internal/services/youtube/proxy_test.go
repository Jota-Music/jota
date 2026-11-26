package youtube

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// register puts a stream in the relay's table for the length of one test.
func register(t *testing.T, id, url, mime string) {
	t.Helper()
	proxyMu.Lock()
	proxyStreams[id] = stream{url: url, mime: mime}
	proxyMu.Unlock()
	t.Cleanup(func() {
		proxyMu.Lock()
		delete(proxyStreams, id)
		proxyMu.Unlock()
	})
}

// The player seeks with a byte range. The proxy must ask googlevideo for that
// same range and hand back its answer untouched: a body that starts where the
// headers do not say is a seek into the wrong bytes, which the player reports
// as the end of the track.
func TestRelayForwardsRange(t *testing.T) {
	body := strings.Repeat("0123456789", 100)
	var gotRange string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		start, end, _, ok := parseRange(gotRange)
		if !ok {
			t.Errorf("upstream got no range: %q", gotRange)
			http.Error(w, "range required", http.StatusBadRequest)
			return
		}
		// googlevideo clamps a range that asks past the end of the stream.
		if end > int64(len(body))-1 {
			end = int64(len(body)) - 1
		}
		w.Header().Set("Content-Type", "audio/mp4")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(body)))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, body[start:end+1])
	}))
	defer upstream.Close()
	register(t, "seek", upstream.URL, "audio/mp4")

	req := httptest.NewRequest(http.MethodGet, "/stream/seek", nil)
	req.Header.Set("Range", "bytes=400-")
	rec := httptest.NewRecorder()
	relay(rec, req)

	if start, _, _, _ := parseRange(gotRange); start != 400 {
		t.Fatalf("upstream got range %q, want it to start at 400", gotRange)
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

// A play costs one upstream request: the slice the player gets is also the
// answer that carries the stream's size. A probe before it doubles the latency
// of every start and every seek, which is what a phone notices first.
func TestRelayAsksTheUpstreamOnce(t *testing.T) {
	var calls int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		start, end, _, _ := parseRange(r.Header.Get("Range"))
		if end > 999 {
			end = 999
		}
		w.Header().Set("Content-Type", "audio/webm")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/1000", start, end))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, strings.Repeat("a", int(end-start+1)))
	}))
	defer upstream.Close()
	register(t, "once", upstream.URL, "audio/webm")

	req := httptest.NewRequest(http.MethodGet, "/stream/once", nil)
	req.Header.Set("Range", "bytes=0-999")
	rec := httptest.NewRecorder()
	relay(rec, req)

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("upstream asked %d times, want 1", got)
	}
	if rec.Code != http.StatusPartialContent || rec.Body.Len() != 1000 {
		t.Fatalf("status = %d, body = %d bytes", rec.Code, rec.Body.Len())
	}
}

// A stream longer than one chunk is fetched as a chain of slices, and the size
// the headers carry came from the first of them.
func TestRelayChainsChunks(t *testing.T) {
	size := 2*chunkSize + 4096
	body := make([]byte, size)
	for i := range body {
		body[i] = byte(i % 251)
	}
	var calls int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		start, end, _, _ := parseRange(r.Header.Get("Range"))
		if end > int64(size)-1 {
			end = int64(size) - 1
		}
		w.Header().Set("Content-Type", "audio/webm")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(body[start : end+1])
	}))
	defer upstream.Close()
	register(t, "chain", upstream.URL, "audio/webm")

	req := httptest.NewRequest(http.MethodGet, "/stream/chain", nil)
	req.Header.Set("Range", "bytes=0-")
	rec := httptest.NewRecorder()
	relay(rec, req)

	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("upstream asked %d times, want 3 for %d bytes", got, size)
	}
	if rec.Body.Len() != size {
		t.Fatalf("body carried %d bytes, want %d", rec.Body.Len(), size)
	}
	if !bytes.Equal(rec.Body.Bytes(), body) {
		t.Fatal("the chained slices do not rebuild the stream")
	}
	if cl := rec.Header().Get("Content-Length"); cl != strconv.Itoa(size) {
		t.Fatalf("content-length = %q, want %d", cl, size)
	}
}

// A HEAD is answered from a one-byte probe: the stream itself is not pulled
// through the relay just so the player can read its length.
func TestRelayHeadProbesOnly(t *testing.T) {
	var gotRange string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		w.Header().Set("Content-Type", "audio/mp4")
		w.Header().Set("Content-Range", "bytes 0-0/5000")
		w.Header().Set("Content-Length", "1")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, "x")
	}))
	defer upstream.Close()
	register(t, "head", upstream.URL, "audio/mp4")

	rec := httptest.NewRecorder()
	relay(rec, httptest.NewRequest(http.MethodHead, "/stream/head", nil))

	if gotRange != "bytes=0-0" {
		t.Fatalf("upstream got range %q, want bytes=0-0", gotRange)
	}
	if cl := rec.Header().Get("Content-Length"); cl != "5000" {
		t.Fatalf("content-length = %q, want the stream's 5000", cl)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("head carried %d bytes", rec.Body.Len())
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
	register(t, "whole", upstream.URL, "audio/webm")

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

// A player reads the index at the end of an MP4 with a suffix range, and
// googlevideo refuses that form (416). The relay has to learn the size and ask
// for the bytes the suffix means: answering the whole stream instead made the
// player read megabytes it did not ask for, then re-fetch what it needed, which
// is the track stopping and loading again right after it started.
func TestRelayResolvesSuffixRange(t *testing.T) {
	body := strings.Repeat("0123456789", 100)
	var first string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange := r.Header.Get("Range")
		if first == "" {
			first = gotRange
		}
		if strings.HasPrefix(gotRange, "bytes=-") {
			http.Error(w, "suffix unsupported", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		start, end, _, ok := parseRange(gotRange)
		if !ok || start < 0 || end < start {
			t.Errorf("upstream got range %q", gotRange)
			http.Error(w, "range required", http.StatusBadRequest)
			return
		}
		if end > int64(len(body))-1 {
			end = int64(len(body)) - 1
		}
		w.Header().Set("Content-Type", "audio/mp4")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(body)))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = io.WriteString(w, body[start:end+1])
	}))
	defer upstream.Close()
	register(t, "suffix", upstream.URL, "audio/mp4")

	req := httptest.NewRequest(http.MethodGet, "/stream/suffix", nil)
	req.Header.Set("Range", "bytes=-100")
	rec := httptest.NewRecorder()
	relay(rec, req)

	if first != "bytes=0-0" {
		t.Fatalf("upstream was asked %q first, want the one-byte size probe", first)
	}
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("status = %d, want 206", rec.Code)
	}
	if cr := rec.Header().Get("Content-Range"); cr != "bytes 900-999/1000" {
		t.Fatalf("content-range = %q, want the last 100 bytes", cr)
	}
	if got := rec.Body.String(); got != body[900:] {
		t.Fatalf("body carried %d bytes, want the last 100", len(got))
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
