package youtube

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// seedVisitor marks the visitor as fresh so getVisitorData serves it from memory
// instead of reaching the homepage.
func seedVisitor(t *testing.T, token string) {
	t.Helper()
	visitorDataMu.Lock()
	origVisitor, origLast, origBlocked := visitorData, lastFetch, blockedUntil
	visitorData, lastFetch, blockedUntil = token, time.Now(), time.Time{}
	visitorDataMu.Unlock()
	t.Cleanup(func() {
		visitorDataMu.Lock()
		visitorData, lastFetch, blockedUntil = origVisitor, origLast, origBlocked
		visitorDataMu.Unlock()
	})
}

func withInnertubeTransport(t *testing.T, fn roundTripFunc) {
	t.Helper()
	orig := innertubeClient
	innertubeClient = &http.Client{Transport: fn}
	t.Cleanup(func() { innertubeClient = orig })
}

func TestRetryRequestRetriesRateLimitAndServerErrors(t *testing.T) {
	seedVisitor(t, "vis-token")

	var calls int32
	withInnertubeTransport(t, func(*http.Request) (*http.Response, error) {
		switch atomic.AddInt32(&calls, 1) {
		case 1:
			return &http.Response{StatusCode: http.StatusTooManyRequests, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
		case 2:
			return &http.Response{StatusCode: http.StatusInternalServerError, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
		default:
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok"))}, nil
		}
	})

	data, err := retryRequest(preferredClient, "https://example.test/player", map[string]any{}, true, 5)
	if err != nil {
		t.Fatalf("retryRequest: %v", err)
	}
	if string(data) != "ok" {
		t.Fatalf("data = %q, want ok", data)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("requests = %d, want 3 (429 and 500 retried)", got)
	}
}

func TestRetryRequestGivesUpAfterRetries(t *testing.T) {
	seedVisitor(t, "vis-token")

	var calls int32
	withInnertubeTransport(t, func(*http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
	})

	if _, err := retryRequest(preferredClient, "https://example.test/player", map[string]any{}, true, 3); err == nil {
		t.Fatal("expected an error once retries are exhausted")
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("requests = %d, want 3", got)
	}
}

func TestDoRequestSendsVisitorHeader(t *testing.T) {
	seedVisitor(t, "vis-xyz")

	var got string
	withInnertubeTransport(t, func(r *http.Request) (*http.Response, error) {
		got = r.Header.Get("X-Goog-Visitor-Id")
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})

	if _, err := retryRequest(preferredClient, "https://example.test/player", map[string]any{}, true, 1); err != nil {
		t.Fatalf("retryRequest: %v", err)
	}
	if got != "vis-xyz" {
		t.Fatalf("X-Goog-Visitor-Id = %q, want vis-xyz", got)
	}
}

func TestGetVisitorDataNegativeCache(t *testing.T) {
	origClient := visitorClient
	defer func() { visitorClient = origClient }()

	apiKeyMu.Lock()
	origKey := apiKey
	apiKeyMu.Unlock()
	visitorDataMu.Lock()
	origVisitor, origLast, origBlocked := visitorData, lastFetch, blockedUntil
	visitorData, lastFetch, blockedUntil = "", time.Time{}, time.Time{}
	visitorDataMu.Unlock()
	_ = visitorBucket.Delete(visitorKey)
	defer func() {
		apiKeyMu.Lock()
		apiKey = origKey
		apiKeyMu.Unlock()
		visitorDataMu.Lock()
		visitorData, lastFetch, blockedUntil = origVisitor, origLast, origBlocked
		visitorDataMu.Unlock()
		_ = visitorBucket.Delete(visitorKey)
	}()

	var hits int32
	// Homepage variant WITHOUT VISITOR_DATA: exercises the negative cache.
	body := `"INNERTUBE_API_KEY":"test_key"`
	visitorClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			atomic.AddInt32(&hits, 1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}

	if _, key, err := getVisitorData(); err != nil || key != "test_key" {
		t.Fatalf("first call: key=%q err=%v", key, err)
	}
	if _, _, err := getVisitorData(); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("homepage fetched %d times, want 1 (negative cache)", got)
	}

	visitorDataMu.Lock()
	lastFetch = time.Now().Add(-visitorTTL - time.Second)
	visitorDataMu.Unlock()
	if _, _, err := getVisitorData(); err != nil {
		t.Fatalf("third call: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("homepage fetched %d times after TTL, want 2", got)
	}
}

func TestGetVisitorBacksOffAfterFailure(t *testing.T) {
	origClient := visitorClient
	defer func() { visitorClient = origClient }()

	visitorDataMu.Lock()
	origVisitor, origLast, origBlocked := visitorData, lastFetch, blockedUntil
	visitorData, lastFetch, blockedUntil = "", time.Time{}, time.Time{}
	visitorDataMu.Unlock()
	_ = visitorBucket.Delete(visitorKey)
	defer func() {
		visitorDataMu.Lock()
		visitorData, lastFetch, blockedUntil = origVisitor, origLast, origBlocked
		visitorDataMu.Unlock()
	}()

	var hits int32
	visitorClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			atomic.AddInt32(&hits, 1)
			return nil, errors.New("blocked")
		}),
	}

	if _, _, err := getVisitorData(); err != nil {
		t.Fatalf("first call: %v", err)
	}
	attempted := atomic.LoadInt32(&hits)
	if attempted == 0 {
		t.Fatal("expected homepage fetch attempts")
	}

	if _, _, err := getVisitorData(); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != attempted {
		t.Fatalf("homepage fetched %d times during backoff, want %d", got, attempted)
	}
}

func TestAdoptVisitorFromResponse(t *testing.T) {
	visitorDataMu.Lock()
	origVisitor, origLast, origBlocked := visitorData, lastFetch, blockedUntil
	visitorData, lastFetch, blockedUntil = "", time.Time{}, time.Now().Add(time.Minute)
	visitorDataMu.Unlock()
	_ = visitorBucket.Delete(visitorKey)
	defer func() {
		visitorDataMu.Lock()
		visitorData, lastFetch, blockedUntil = origVisitor, origLast, origBlocked
		visitorDataMu.Unlock()
		_ = visitorBucket.Delete(visitorKey)
	}()

	adoptVisitor([]byte(`{"responseContext":{"visitorData":"token-123"}}`))

	visitorDataMu.RLock()
	got, blocked := visitorData, blockedUntil
	visitorDataMu.RUnlock()
	if got != "token-123" {
		t.Fatalf("visitorData = %q, want token-123", got)
	}
	if !blocked.IsZero() {
		t.Fatal("expected a successful response to clear the backoff")
	}
	if stored, err := visitorBucket.GetString(visitorKey); err != nil || stored != "token-123" {
		t.Fatalf("persisted token = %q, %v; want token-123", stored, err)
	}
}

func TestGetVisitorDataResumesPersistedToken(t *testing.T) {
	origClient := visitorClient
	defer func() { visitorClient = origClient }()

	visitorDataMu.Lock()
	origVisitor, origLast, origBlocked := visitorData, lastFetch, blockedUntil
	visitorData, lastFetch, blockedUntil = "", time.Time{}, time.Time{}
	visitorDataMu.Unlock()
	_ = visitorBucket.Delete(visitorKey)
	defer func() {
		visitorDataMu.Lock()
		visitorData, lastFetch, blockedUntil = origVisitor, origLast, origBlocked
		visitorDataMu.Unlock()
		_ = visitorBucket.Delete(visitorKey)
	}()

	if err := visitorBucket.SetString(visitorKey, "persisted-token", visitorTTL); err != nil {
		t.Fatalf("seed bucket: %v", err)
	}

	var hits int32
	visitorClient = &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			atomic.AddInt32(&hits, 1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}),
	}

	token, _, err := getVisitorData()
	if err != nil {
		t.Fatalf("getVisitorData: %v", err)
	}
	if token != "persisted-token" {
		t.Fatalf("token = %q, want persisted-token", token)
	}
	if got := atomic.LoadInt32(&hits); got != 0 {
		t.Fatalf("homepage fetched %d times, want 0 (resumed from disk)", got)
	}
}

func TestVisitorRedirectStopsAtGoogle(t *testing.T) {
	req := func(host string) *http.Request {
		r, _ := http.NewRequest("GET", "https://"+host+"/", nil)
		return r
	}

	cases := []struct {
		name string
		req  *http.Request
		via  []*http.Request
		want error
	}{
		{"google bot check", req("www.google.com"), nil, http.ErrUseLastResponse},
		{"google apex", req("google.com"), nil, http.ErrUseLastResponse},
		{"youtube homepage", req("www.youtube.com"), nil, nil},
		{"notgoogle is not google", req("notgoogle.com"), nil, nil},
		{"too many redirects", req("www.youtube.com"), make([]*http.Request, 3), http.ErrUseLastResponse},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := visitorRedirect(c.req, c.via); got != c.want {
				t.Fatalf("visitorRedirect() = %v, want %v", got, c.want)
			}
		})
	}
}
