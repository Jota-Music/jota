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

const channelSearchFixture = `{
  "contents": {
    "sectionListRenderer": {
      "contents": [{
        "itemSectionRenderer": {
          "contents": [
            {
              "compactChannelRenderer": {
                "channelId": "UCq19-LqvG35A-30oyAiPiqA",
                "displayName": { "runs": [{ "text": "Radiohead" }] },
                "videoCountText": { "runs": [{ "text": "5.4M subscribers" }] },
                "subscriberCountText": { "runs": [{ "text": "@Radiohead" }] },
                "thumbnail": {
                  "thumbnails": [
                    { "url": "//yt3.ggpht.com/s88=s88-c-k-c0x00ffffff-no-rj", "width": 88, "height": 88 },
                    { "url": "//yt3.ggpht.com/s176=s176-c-k-c0x00ffffff-no-rj", "width": 176, "height": 176 }
                  ]
                }
              }
            },
            { "compactChannelRenderer": { "channelId": "" } }
          ]
        }
      }]
    }
  }
}`

func TestChannelSearchParsing(t *testing.T) {
	var res searchResponse
	if err := json.Unmarshal([]byte(channelSearchFixture), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	channels := extractChannels(res)
	if len(channels) != 1 {
		t.Fatalf("channels = %d, want 1", len(channels))
	}
	c := channels[0]
	if c.Id != "UCq19-LqvG35A-30oyAiPiqA" || c.Name != "Radiohead" {
		t.Fatalf("channel = %+v", c)
	}
	if c.Avatar != "https://yt3.ggpht.com/s176=s176-c-k-c0x00ffffff-no-rj" {
		t.Fatalf("avatar = %q, want the largest thumbnail over https", c.Avatar)
	}
}
