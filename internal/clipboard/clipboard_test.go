package clipboard

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// The answer arrives on the platform's loop, after the call returned: this is
// what keeps a slow clipboard owner from freezing the caller.
func TestTextTakesTheLateAnswer(t *testing.T) {
	restore := swap(func(deliver func(string)) {
		go func() {
			time.Sleep(20 * time.Millisecond)
			deliver("https://example.test/track/1")
		}()
	})
	defer restore()

	got, err := Text()
	if err != nil {
		t.Fatalf("Text: %v", err)
	}
	if got != "https://example.test/track/1" {
		t.Fatalf("got %q", got)
	}
}

// An owner that never answers must not hold the caller: an empty answer and a
// timeout mean the same thing to the page.
func TestTextGivesUpWithoutAnAnswer(t *testing.T) {
	restore := swap(func(func(string)) {})
	defer restore()

	start := time.Now()
	got, err := Text()
	if err != nil {
		t.Fatalf("Text: %v", err)
	}
	if got != "" {
		t.Fatalf("got %q from a silent owner", got)
	}
	if elapsed := time.Since(start); elapsed > Timeout+200*time.Millisecond {
		t.Fatalf("read outlived its budget: %s", elapsed)
	}
}

// A stale answer must not resolve the next request.
func TestTextIgnoresAStaleAnswer(t *testing.T) {
	// The delivery slot crosses goroutines, so it travels on a channel: the
	// callback runs inside the request Text started.
	delivers := make(chan func(string), 1)
	var calls atomic.Int32
	restore := swap(func(deliver func(string)) {
		if calls.Add(1) == 1 {
			delivers <- deliver
			return
		}
		deliver("fresh")
	})
	defer restore()

	if _, err := Text(); err != nil {
		t.Fatalf("first: %v", err)
	}
	stale := <-delivers
	stale("stale")
	got, err := Text()
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if got != "fresh" {
		t.Fatalf("stale answer won: %q", got)
	}
}

func TestTextReportsUnsupported(t *testing.T) {
	restore := swap(nil)
	defer restore()

	if _, err := Text(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("want ErrUnsupported, got %v", err)
	}
}

func swap(fn func(func(string))) func() {
	old := request
	request = fn
	return func() { request = old }
}
