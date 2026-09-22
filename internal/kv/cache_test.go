package kv

import "testing"

func TestCachedFetchesOnce(t *testing.T) {
	openTemp(t)
	b := UseBucket("cached-test")

	calls := 0
	fetch := func() (string, error) {
		calls++
		return "value", nil
	}

	for i := 0; i < 2; i++ {
		got, err := Cached(b, "k", fetch)
		if err != nil || got != "value" {
			t.Fatalf("Cached = %q, %v", got, err)
		}
	}
	if calls != 1 {
		t.Fatalf("fetch called %d times, want 1", calls)
	}
}

func TestRefreshableDetectsChange(t *testing.T) {
	openTemp(t)
	b := UseBucket("cached-test")

	value := "one"
	if _, err := Cached(b, "k", func() (string, error) { return value, nil }); err != nil {
		t.Fatalf("Cached: %v", err)
	}

	if got := Refreshable(b, "k", func() (string, error) { return value, nil }); got {
		t.Fatalf("unchanged refresh rewrote the entry")
	}

	value = "two"
	if !Refreshable(b, "k", func() (string, error) { return value, nil }) {
		t.Fatalf("changed refresh did not rewrite the entry")
	}

	got, err := Cached(b, "k", func() (string, error) { return value, nil })
	if err != nil || got != "two" {
		t.Fatalf("Cached = %q, %v", got, err)
	}
}
