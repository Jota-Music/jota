package youtube

import (
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/music"
)

func TestSetYoutubeIdInvalidatesAudioCache(t *testing.T) {
	const song = "test-set-youtube-id"
	future := time.Now().Add(time.Hour).Unix()
	_ = audioBucket.SetObject("spotify:"+song, music.Audio{Url: "https://example.com/a", VideoID: "old_video1", ExpireAt: future}, time.Hour)
	_ = audioBucket.SetObject("old_video1", music.Audio{Url: "https://example.com/a", VideoID: "old_video1", ExpireAt: future}, time.Hour)
	_ = youtubeSourceBucket.SetString(song, "old_video1")

	svc := NewService()
	if err := svc.SetYoutubeId(song, "new_video1"); err != nil {
		t.Fatalf("SetYoutubeId: %v", err)
	}
	if got := cachedAudioBySong(song); got != nil {
		t.Fatalf("song cache not invalidated: %+v", got)
	}
	if got := cachedAudio("old_video1"); got != nil {
		t.Fatalf("youtube cache not invalidated: %+v", got)
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
