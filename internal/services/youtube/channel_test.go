package youtube

import (
	"encoding/json"
	"testing"
)

const channelFixture = `{
  "contents": {
    "singleColumnBrowseResultsRenderer": {
      "tabs": [{
        "tabRenderer": {
          "title": "Playlists",
          "content": {
            "sectionListRenderer": {
              "contents": [{
                "itemSectionRenderer": {
                  "contents": [
                    { "compactPlaylistRenderer": {
                        "playlistId": "PLabc123",
                        "title": { "runs": [{ "text": "Lo-fi Beats" }] },
                        "shortBylineText": { "runs": [{ "text": "Chill Channel" }] },
                        "thumbnail": { "thumbnails": [
                          { "url": "https://i.ytimg.com/vi/x/hq720.jpg" }
                        ]}
                    }},
                    { "compactPlaylistRenderer": { "playlistId": "" } }
                  ],
                  "continuations": [{
                    "nextContinuationData": { "continuation": "TOKEN1" }
                  }]
                }
              }]
            }
          }
        }
      }]
    }
  }
}`

const channelContinuationFixture = `{
  "continuationContents": {
    "itemSectionContinuation": {
      "contents": [
        { "compactPlaylistRenderer": {
            "playlistId": "PLdef456",
            "title": { "runs": [{ "text": "Second Mix" }] },
            "shortBylineText": { "runs": [{ "text": "Chill Channel" }] },
            "thumbnail": { "thumbnails": [{ "url": "https://i.ytimg.com/vi/y/mqdefault.jpg" }] }
        }}
      ]
    }
  }
}`

const channelVideosFixture = `{
  "contents": {
    "singleColumnBrowseResultsRenderer": {
      "tabs": [{
        "tabRenderer": {
          "title": "Videos",
          "content": {
            "sectionListRenderer": {
              "contents": [{
                "itemSectionRenderer": {
                  "contents": [
                    { "compactVideoRenderer": {
                        "videoId": "cNYSOHWbkJU",
                        "title": { "runs": [{ "text": "Earthquake by Mount Fuji" }] },
                        "lengthText": { "runs": [{ "text": "32:16" }] },
                        "thumbnail": { "thumbnails": [
                          { "url": "https://i.ytimg.com/vi/cNYSOHWbkJU/hq720.jpg" }
                        ]}
                    }},
                    { "compactVideoRenderer": { "videoId": "" } }
                  ]
                }
              }]
            }
          }
        }
      }]
    }
  }
}`

func TestChannelPlaylistsParsing(t *testing.T) {
	var res channelBrowseResponse
	if err := json.Unmarshal([]byte(channelFixture), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	section, ok := res.section()
	if !ok {
		t.Fatal("expected a playlists section")
	}
	if got := section.next(); got != "TOKEN1" {
		t.Fatalf("continuation = %q", got)
	}

	playlists := section.summaries()
	if len(playlists) != 1 {
		t.Fatalf("playlists = %d, want 1", len(playlists))
	}
	p := playlists[0]
	if p.Id != "youtube:PLabc123" || p.Name != "Lo-fi Beats" || p.Subtitle != "Chill Channel" {
		t.Fatalf("playlist = %+v", p)
	}
	if p.Cover != "https://i.ytimg.com/vi/x/hqdefault.jpg" {
		t.Fatalf("cover = %q", p.Cover)
	}
}

func TestChannelPlaylistsContinuation(t *testing.T) {
	var cont channelContinuationResponse
	if err := json.Unmarshal([]byte(channelContinuationFixture), &cont); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	playlists := cont.ContinuationContents.ItemSectionContinuation.summaries()
	if len(playlists) != 1 {
		t.Fatalf("playlists = %d, want 1", len(playlists))
	}
	if playlists[0].Id != "youtube:PLdef456" || playlists[0].Name != "Second Mix" {
		t.Fatalf("playlist = %+v", playlists[0])
	}
}

func TestResolveChannelIDPassthrough(t *testing.T) {
	const id = "UC-lHJZR3Gqxm24_Vd_AJ5Yw"
	got, err := resolveChannelID(id)
	if err != nil || got != id {
		t.Fatalf("resolveChannelID(%q) = %q/%v", id, got, err)
	}
}

func TestChannelVideosParsing(t *testing.T) {
	var res channelBrowseResponse
	if err := json.Unmarshal([]byte(channelVideosFixture), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	section, ok := res.section()
	if !ok {
		t.Fatal("expected a videos section")
	}

	videos := section.videos("UC-lHJZR3Gqxm24_Vd_AJ5Yw", "Chill Channel")
	if len(videos) != 1 {
		t.Fatalf("videos = %d, want 1", len(videos))
	}
	v := videos[0]
	if v.Id != "youtube:cNYSOHWbkJU" || v.Name != "Earthquake by Mount Fuji" {
		t.Fatalf("video = %+v", v)
	}
	if v.Duration != 1936 {
		t.Fatalf("duration = %d, want 1936", v.Duration)
	}
	if len(v.Artists) != 1 || v.Artists[0].Id != "UC-lHJZR3Gqxm24_Vd_AJ5Yw" ||
		v.Artists[0].Source != "youtube" || v.Artists[0].Name != "Chill Channel" {
		t.Fatalf("artists = %+v", v.Artists)
	}
}
