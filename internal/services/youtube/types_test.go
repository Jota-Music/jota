package youtube

import "testing"

func TestCapThumbnail(t *testing.T) {
	cases := map[string]string{
		"https://i.ytimg.com/vi/fOT0BUpITw8/maxresdefault.jpg":    "https://i.ytimg.com/vi/fOT0BUpITw8/hqdefault.jpg",
		"https://i.ytimg.com/vi/fOT0BUpITw8/hq720.jpg":            "https://i.ytimg.com/vi/fOT0BUpITw8/hqdefault.jpg",
		"https://i.ytimg.com/vi/fOT0BUpITw8/sddefault.webp":       "https://i.ytimg.com/vi/fOT0BUpITw8/hqdefault.webp",
		"https://i.ytimg.com/vi/fOT0BUpITw8/hqdefault.jpg":        "https://i.ytimg.com/vi/fOT0BUpITw8/hqdefault.jpg",
		"https://i.ytimg.com/vi/fOT0BUpITw8/mqdefault.jpg":        "https://i.ytimg.com/vi/fOT0BUpITw8/mqdefault.jpg",
		"https://i.ytimg.com/vi/fOT0BUpITw8/default.jpg":          "https://i.ytimg.com/vi/fOT0BUpITw8/default.jpg",
		"https://i.ytimg.com/vi/fOT0BUpITw8/hq720.jpg?sqp=abc123": "https://i.ytimg.com/vi/fOT0BUpITw8/hqdefault.jpg?sqp=abc123",
		"https://example.com/other/hq720.jpg":                     "https://example.com/other/hq720.jpg",
	}
	for in, want := range cases {
		if got := capThumbnail(in); got != want {
			t.Errorf("capThumbnail(%q) = %q, want %q", in, got, want)
		}
	}
}
