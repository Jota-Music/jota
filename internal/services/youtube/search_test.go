package youtube

import (
	"encoding/json"
	"testing"
)

const videoSearchFixture = `{
  "contents": {
    "sectionListRenderer": {
      "contents": [{
        "itemSectionRenderer": {
          "contents": [
            {
              "compactVideoRenderer": {
                "videoId": "dQw4w9WgXcQ",
                "title": { "runs": [{ "text": "Never Gonna Give You Up" }] },
                "shortBylineText": { "runs": [{ "text": "Rick Astley" }] }
              }
            },
            { "compactVideoRenderer": { "videoId": "" } }
          ]
        }
      }]
    }
  }
}`

func TestVideoSearchParsing(t *testing.T) {
	var sr searchResponse
	if err := json.Unmarshal([]byte(videoSearchFixture), &sr); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	videos := extractVideos(sr)
	if len(videos) != 1 {
		t.Fatalf("videos = %d, want 1", len(videos))
	}
	v := videos[0]
	if v.ID != "dQw4w9WgXcQ" || v.Title != "Never Gonna Give You Up" || v.Author != "Rick Astley" {
		t.Fatalf("video = %+v", v)
	}
}
