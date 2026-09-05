package youtube

import "testing"

func TestGetAudioURL(t *testing.T) {
	url, err := GetAudioURL("E9s9BNZFQLA")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if url == "" || len(url) < 50 {
		t.Fatalf("invalid URL: %s", url)
	}
}