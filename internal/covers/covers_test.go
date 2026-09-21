package covers

import (
	"image"
	"testing"
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
