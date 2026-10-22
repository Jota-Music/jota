package covers

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestAllowed(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		ok   bool
		want string
	}{
		{"spotify", "https://i.scdn.co/image/abc", true, "https://i.scdn.co/image/abc"},
		{"mosaic", "https://mosaic.scdn.co/640/abc", true, "https://mosaic.scdn.co/640/abc"},
		{"avatar", "https://lh3.googleusercontent.com/a/x", true, "https://lh3.googleusercontent.com/a/x"},
		{"http rejected", "http://i.scdn.co/image/abc", false, ""},
		{"unknown host", "https://evil.example.com/a.jpg", false, ""},
		{"empty", "", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := allowed(tc.raw)
			if ok != tc.ok {
				t.Fatalf("allowed(%q) ok = %v, want %v", tc.raw, ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("allowed(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestClampWidth(t *testing.T) {
	cases := map[string]int{
		"":      defaultWidth,
		"0":     defaultWidth,
		"-5":    defaultWidth,
		"abc":   defaultWidth,
		"1":     minWidth,
		"100":   100,
		"10000": maxWidth,
	}
	for raw, want := range cases {
		if got := clampWidth(raw); got != want {
			t.Fatalf("clampWidth(%q) = %d, want %d", raw, got, want)
		}
	}
}

func TestBucketWidth(t *testing.T) {
	cases := map[int]int{
		1:   80,
		80:  80,
		81:  128,
		128: 128,
		129: 160,
		160: 160,
		161: 320,
		320: 320,
		321: 480,
		384: 480,
		480: 480,
		481: 640,
		640: 640,
	}
	for width, want := range cases {
		if got := bucketWidth(width); got != want {
			t.Fatalf("bucketWidth(%d) = %d, want %d", width, got, want)
		}
	}
}

func TestLRUEvictsLeastRecentlyUsed(t *testing.T) {
	cache := newLRU(10)

	cache.put("a", make([]byte, 4))
	cache.put("b", make([]byte, 4))
	if _, ok := cache.get("a"); !ok {
		t.Fatal("expected a to be cached")
	}

	cache.put("c", make([]byte, 4))

	if _, ok := cache.get("b"); ok {
		t.Fatal("expected b to be evicted")
	}
	if _, ok := cache.get("a"); !ok {
		t.Fatal("expected a to survive eviction")
	}
	if _, ok := cache.get("c"); !ok {
		t.Fatal("expected c to be cached")
	}
}

func TestLRUReplacesSameKey(t *testing.T) {
	cache := newLRU(10)
	cache.put("a", make([]byte, 4))
	cache.put("a", make([]byte, 6))

	if cache.bytes != 6 {
		t.Fatalf("bytes = %d, want 6", cache.bytes)
	}
}

func TestResize(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 200, 100))

	small := resize(src, 100)
	if b := small.Bounds(); b.Dx() != 100 || b.Dy() != 50 {
		t.Fatalf("resize to 100 = %dx%d, want 100x50", b.Dx(), b.Dy())
	}

	same := resize(src, 200)
	if b := same.Bounds(); b.Dx() != 200 || b.Dy() != 100 {
		t.Fatalf("resize to source width = %dx%d, want 200x100", b.Dx(), b.Dy())
	}

	larger := resize(src, 400)
	if b := larger.Bounds(); b.Dx() != 200 || b.Dy() != 100 {
		t.Fatalf("resize above source = %dx%d, want unchanged 200x100", b.Dx(), b.Dy())
	}
}

// gateTripper answers every request with the same in-memory image, waits on
// gate before answering, and counts how many requests reached upstream.
type gateTripper struct {
	mu    sync.Mutex
	hits  int
	gate  chan struct{}
	image []byte
}

func (t *gateTripper) RoundTrip(*http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.hits++
	t.mu.Unlock()
	if t.gate != nil {
		<-t.gate
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(t.image)),
	}, nil
}

func (t *gateTripper) count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.hits
}

func pngSource(tb testing.TB, size int) []byte {
	tb.Helper()
	var buf bytes.Buffer
	src := image.NewRGBA(image.Rect(0, 0, size, size))
	if err := png.Encode(&buf, src); err != nil {
		tb.Fatal(err)
	}
	return buf.Bytes()
}

func TestConcurrentRequestsShareRender(t *testing.T) {
	tripper := &gateTripper{gate: make(chan struct{}), image: pngSource(t, 640)}
	server := New()
	server.client = &http.Client{Transport: tripper}

	const callers = 8
	var wg sync.WaitGroup
	bodies := make([][]byte, callers)
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, path+"?u=https://i.scdn.co/image/a&w=320", nil)
			rec := httptest.NewRecorder()
			server.Middleware(http.NotFoundHandler()).ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want 200", rec.Code)
			}
			bodies[i] = rec.Body.Bytes()
		}()
	}

	// Give every caller time to join the in-flight render, then let it finish.
	time.Sleep(50 * time.Millisecond)
	close(tripper.gate)
	wg.Wait()

	if got := tripper.count(); got != 1 {
		t.Fatalf("upstream requests = %d, want 1", got)
	}
	for i, body := range bodies {
		if !bytes.Equal(body, bodies[0]) {
			t.Fatalf("caller %d got a different cover", i)
		}
	}
}

func BenchmarkServeScrollingCovers(b *testing.B) {
	tripper := &gateTripper{image: pngSource(b, 640)}
	s := New()
	s.client = &http.Client{Transport: tripper}
	handler := s.Middleware(http.NotFoundHandler())
	widths := []int{80, 160, 320}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			i++
			req := httptest.NewRequest(http.MethodGet,
				fmt.Sprintf("%s?u=https://i.scdn.co/image/%d&w=%d", path, i%200, widths[i%3]), nil)
			handler.ServeHTTP(httptest.NewRecorder(), req)
		}
	})
}
