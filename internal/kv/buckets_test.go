package kv

import (
	"testing"
	"time"
)

func openTemp(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	Close()
	if err := EnsureStarted(); err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(Close)
}

func TestBucketRoundtrip(t *testing.T) {
	openTemp(t)
	b := UseBucket("test")

	if _, err := b.GetString("missing"); err != ErrKeyNotFound {
		t.Fatalf("missing key: want ErrKeyNotFound, got %v", err)
	}

	if err := b.SetString("s", "hello"); err != nil {
		t.Fatalf("SetString: %v", err)
	}
	if got, err := b.GetString("s"); err != nil || got != "hello" {
		t.Fatalf("GetString = %q, %v", got, err)
	}

	type obj struct {
		A int    `json:"a"`
		B string `json:"b"`
	}
	if err := b.SetObject("o", obj{A: 1, B: "x"}, time.Hour); err != nil {
		t.Fatalf("SetObject: %v", err)
	}
	var out obj
	if err := b.GetObject("o", &out); err != nil || out.A != 1 || out.B != "x" {
		t.Fatalf("GetObject = %+v, %v", out, err)
	}

	if err := b.Delete("s"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := b.GetString("s"); err != ErrKeyNotFound {
		t.Fatalf("after Delete: want ErrKeyNotFound, got %v", err)
	}
	if err := b.Delete("nope"); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

func TestBucketsAreNamespaced(t *testing.T) {
	openTemp(t)
	a := UseBucket("a")
	b := UseBucket("b")

	if err := a.SetString("k", "va"); err != nil {
		t.Fatal(err)
	}
	if err := b.SetString("k", "vb"); err != nil {
		t.Fatal(err)
	}
	if got, _ := a.GetString("k"); got != "va" {
		t.Fatalf("bucket a = %q", got)
	}
	if got, _ := b.GetString("k"); got != "vb" {
		t.Fatalf("bucket b = %q", got)
	}
}
