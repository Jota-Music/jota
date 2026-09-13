package youtube

import (
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

func TestGetVisitorDataNegativeCache(t *testing.T) {
	origClient := http.DefaultClient
	defer func() { http.DefaultClient = origClient }()

	apiKeyMu.Lock()
	origKey := apiKey
	apiKeyMu.Unlock()
	visitorDataMu.Lock()
	origVisitor, origLast := visitorData, lastFetch
	visitorData, lastFetch = "", time.Time{}
	visitorDataMu.Unlock()
	defer func() {
		apiKeyMu.Lock()
		apiKey = origKey
		apiKeyMu.Unlock()
		visitorDataMu.Lock()
		visitorData, lastFetch = origVisitor, origLast
		visitorDataMu.Unlock()
	}()

	var hits int32
	// Homepage variant WITHOUT VISITOR_DATA: exercises the negative cache.
	body := `"INNERTUBE_API_KEY":"test_key"`
	http.DefaultClient = &http.Client{
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
