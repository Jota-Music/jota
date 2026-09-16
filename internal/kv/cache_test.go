package kv

import (
	"testing"
	"time"
)

func TestCachedFetchesOnce(t *testing.T) {
	openTemp(t)
	b := UseBucket("cached-test")

	calls := 0
	fetch := func() (string, error) {
		calls++
		return "value", nil
	}

	for i := 0; i < 2; i++ {
		got, err := Cached(b, "k", time.Hour, fetch)
		if err != nil || got != "value" {
			t.Fatalf("Cached = %q, %v", got, err)
		}
	}
	if calls != 1 {
		t.Fatalf("fetch called %d times, want 1", calls)
	}
}
