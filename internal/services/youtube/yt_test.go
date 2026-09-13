package youtube

import (
	"testing"
	"time"
)

func TestSetYoutubeIdInvalidatesAudioCache(t *testing.T) {
	const song = "test-set-youtube-id"
	future := time.Now().Add(time.Hour).Unix()
	_ = audioBucket.SetObject("spotify:"+song, Audio{Url: "https://example.com/a", VideoID: "old_video1", ExpireAt: future}, time.Hour)
	_ = audioBucket.SetObject("old_video1", Audio{Url: "https://example.com/a", VideoID: "old_video1", ExpireAt: future}, time.Hour)
	_ = youtubeSourceBucket.SetString(song, "old_video1")

	if err := SetYoutubeId(song, "new_video1"); err != nil {
		t.Fatalf("SetYoutubeId: %v", err)
	}
	if got := GetCachedAudioBySpotifyId(song); got != nil {
		t.Fatalf("spotify cache not invalidated: %+v", got)
	}
	if got := GetCachedAudioByYoutubeId("old_video1"); got != nil {
		t.Fatalf("youtube cache not invalidated: %+v", got)
	}
}

func TestGetAudioURL(t *testing.T) {
	url, err := GetAudioURL("E9s9BNZFQLA")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if url == "" || len(url) < 50 {
		t.Fatalf("invalid URL: %s", url)
	}
}
