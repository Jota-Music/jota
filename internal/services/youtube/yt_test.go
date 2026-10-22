package youtube

import (
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/music"
)

func TestSetYoutubeIdInvalidatesAudioCache(t *testing.T) {
	const song = "test-set-youtube-id"
	future := time.Now().Add(time.Hour).Unix()
	_ = audioBucket.SetObject("spotify:"+song, music.Audio{Url: "https://example.com/a", VideoID: "old_video01", ExpireAt: future}, time.Hour)
	_ = audioBucket.SetObject("old_video01", music.Audio{Url: "https://example.com/a", VideoID: "old_video01", ExpireAt: future}, time.Hour)
	_ = youtubeSourceBucket.SetString(song, "old_video01")

	svc := NewService()
	if err := svc.SetYoutubeId(song, "new_video01"); err != nil {
		t.Fatalf("SetYoutubeId: %v", err)
	}
	if got := cachedAudioBySong(song); got != nil {
		t.Fatalf("song cache not invalidated: %+v", got)
	}
	if got := cachedAudio("old_video01"); got != nil {
		t.Fatalf("youtube cache not invalidated: %+v", got)
	}
}

func TestSetYoutubeIdStoresOnlyTheId(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ": "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ":                "dQw4w9WgXcQ",
		"https://www.youtube.com/shorts/dQw4w9WgXcQ":  "dQw4w9WgXcQ",
		"youtube:dQw4w9WgXcQ":                         "dQw4w9WgXcQ",
	}

	for input, want := range cases {
		const song = "test-set-youtube-id-url"
		_ = youtubeSourceBucket.Delete(song)

		if err := NewService().SetYoutubeId(song, input); err != nil {
			t.Fatalf("SetYoutubeId(%q): %v", input, err)
		}
		if got := NewService().YoutubeId(song); got != want {
			t.Fatalf("YoutubeId = %q, want %q", got, want)
		}
	}
}

func TestSetYoutubeIdRejectsInvalid(t *testing.T) {
	if err := NewService().SetYoutubeId("test-invalid-youtube", "not a video"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestYoutubeIdLookup(t *testing.T) {
	const song = "test-youtube-id-lookup"
	_ = youtubeSourceBucket.SetString(song, "video1")

	svc := NewService()
	if got := svc.YoutubeId(song); got != "video1" {
		t.Fatalf("YoutubeId = %q, want %q", got, "video1")
	}
	if got := svc.YoutubeId("unknown-song"); got != "" {
		t.Fatalf("YoutubeId = %q, want empty", got)
	}
}

func TestGetAudioURL(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: reaches YouTube")
	}
	url, client, err := audioURL("E9s9BNZFQLA")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if url == "" || len(url) < 50 {
		t.Fatalf("invalid URL: %s", url)
	}
	if client.Name == "" {
		t.Fatalf("expected client name, got empty")
	}
}
