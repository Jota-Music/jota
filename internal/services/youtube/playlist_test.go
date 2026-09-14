package youtube

import (
	"encoding/json"
	"testing"
)

const browseFixture = `{
  "header": {
    "playlistHeaderRenderer": {
      "title": { "runs": [{ "text": "Popular Music Videos" }] },
      "ownerText": { "runs": [{ "text": "Music" }] },
      "playlistHeaderBanner": {
        "heroPlaylistThumbnailRenderer": {
          "thumbnail": { "thumbnails": [
            { "url": "https://i.ytimg.com/vi/fOT0BUpITw8/mqdefault.jpg", "width": 320, "height": 180 }
          ]}
        }
      }
    }
  },
  "contents": {
    "singleColumnBrowseResultsRenderer": {
      "tabs": [{
        "tabRenderer": {
          "content": {
            "sectionListRenderer": {
              "contents": [{
                "playlistVideoListRenderer": {
                  "contents": [
                    { "playlistVideoRenderer": {
                        "videoId": "fOT0BUpITw8",
                        "title": { "runs": [{ "text": "Song One" }] },
                        "shortBylineText": { "runs": [{ "text": "Artist One" }] },
                        "lengthSeconds": "235",
                        "thumbnail": { "thumbnails": [
                          { "url": "https://i.ytimg.com/vi/fOT0BUpITw8/default.jpg" },
                          { "url": "https://i.ytimg.com/vi/fOT0BUpITw8/mqdefault.jpg" }
                        ]}
                    }},
                    { "playlistVideoRenderer": { "videoId": "", "isPlayable": true } },
                    { "playlistVideoRenderer": {
                        "videoId": "Pr1v4t3Ab1",
                        "title": { "runs": [{ "text": "Private" }] },
                        "isPlayable": false
                    }}
                  ],
                  "continuations": [
                    { "nextContinuationData": { "continuation": "TOKEN1" } }
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

const continuationFixture = `{
  "continuationContents": {
    "playlistVideoListContinuation": {
      "contents": [
        { "playlistVideoRenderer": {
            "videoId": "vid2ABCDEFG",
            "title": { "runs": [{ "text": "Song Two" }] },
            "shortBylineText": { "runs": [{ "text": "Artist Two" }] },
            "lengthSeconds": 180,
            "thumbnail": { "thumbnails": [{ "url": "https://img/2.jpg" }] }
        }}
      ]
    }
  }
}`

func TestBrowseParsing(t *testing.T) {
	var res playlistBrowseResponse
	if err := json.Unmarshal([]byte(browseFixture), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	header, ok := res.headerRenderer()
	if !ok {
		t.Fatal("expected header")
	}
	if got := header.Title.first(); got != "Popular Music Videos" {
		t.Fatalf("title = %q", got)
	}
	cover := header.Banner.HeroPlaylistThumbnailRenderer.Thumbnail.url()
	if cover != "https://i.ytimg.com/vi/fOT0BUpITw8/mqdefault.jpg" {
		t.Fatalf("cover = %q", cover)
	}

	list, ok := res.firstList()
	if !ok {
		t.Fatal("expected list")
	}
	if token := list.next(); token != "TOKEN1" {
		t.Fatalf("token = %q", token)
	}

	songs := extractSongs(list)
	if len(songs) != 1 {
		t.Fatalf("songs = %d, want 1", len(songs))
	}

	song := songs[0]
	if song.Id != "youtube:fOT0BUpITw8" {
		t.Fatalf("id = %q", song.Id)
	}
	if song.YoutubeId != "fOT0BUpITw8" {
		t.Fatalf("youtubeId = %q", song.YoutubeId)
	}
	if song.Name != "Song One" {
		t.Fatalf("name = %q", song.Name)
	}
	if song.Duration != 235 {
		t.Fatalf("duration = %d, want 235", song.Duration)
	}
	if len(song.Artists) != 1 || song.Artists[0].Name != "Artist One" {
		t.Fatalf("artists = %+v", song.Artists)
	}
	if len(song.Album.Covers) != 1 || song.Album.Covers[0] != "https://i.ytimg.com/vi/fOT0BUpITw8/mqdefault.jpg" {
		t.Fatalf("covers = %+v", song.Album.Covers)
	}
}

func TestContinuationParsing(t *testing.T) {
	var res playlistContinuationResponse
	if err := json.Unmarshal([]byte(continuationFixture), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	list := res.ContinuationContents.PlaylistVideoListContinuation
	songs := extractSongs(list)
	if len(songs) != 1 {
		t.Fatalf("songs = %d, want 1", len(songs))
	}
	if songs[0].Name != "Song Two" || songs[0].Duration != 180 {
		t.Fatalf("song = %+v", songs[0])
	}
}

const searchFixture = `{
  "contents": {
    "sectionListRenderer": {
      "contents": [{
        "itemSectionRenderer": {
          "contents": [
            { "compactPlaylistRenderer": {
                "playlistId": "PLabc123",
                "title": { "runs": [{ "text": "Corridos Mix" }] },
                "shortBylineText": { "runs": [{ "text": "Revive Music" }] },
                "thumbnail": { "thumbnails": [
                  { "url": "https://i.ytimg.com/vi/x/mqdefault.jpg" },
                  { "url": "https://i.ytimg.com/vi/x/hq720.jpg" }
                ]}
            }},
            { "compactPlaylistRenderer": { "playlistId": "" } }
          ]
        }
      }]
    }
  }
}`

func TestPlaylistSearchParsing(t *testing.T) {
	var res playlistSearchResponse
	if err := json.Unmarshal([]byte(searchFixture), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	playlists := extractPlaylists(res)
	if len(playlists) != 1 {
		t.Fatalf("playlists = %d, want 1", len(playlists))
	}
	p := playlists[0]
	if p.Id != "youtube:PLabc123" {
		t.Fatalf("id = %q", p.Id)
	}
	if p.Name != "Corridos Mix" || p.Subtitle != "Revive Music" {
		t.Fatalf("name/subtitle = %q/%q", p.Name, p.Subtitle)
	}
	if p.Cover != "https://i.ytimg.com/vi/x/hq720.jpg" {
		t.Fatalf("cover = %q", p.Cover)
	}
}

func TestNormalizePlaylistId(t *testing.T) {
	cases := map[string]string{
		"PLabc":             "PLabc",
		"youtube:PLabc":     "PLabc",
		"VLPLabc":           "PLabc",
		" youtube:VLPLabc ": "PLabc",
	}
	for in, want := range cases {
		if got := normalizePlaylistId(in); got != want {
			t.Fatalf("normalize(%q) = %q, want %q", in, got, want)
		}
	}
}
