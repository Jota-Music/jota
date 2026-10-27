package youtube

import (
	"encoding/json"
	"errors"
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

	songs := extractSongs(list, "Popular Music Videos")
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
	if song.Album.Title != "Popular Music Videos" {
		t.Fatalf("album = %q", song.Album.Title)
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

func TestVideoToSongDropsUnavailable(t *testing.T) {
	unplayable := false
	if _, ok := videoToSong(playlistVideoRenderer{VideoId: "dQw4w9WgXcQ", IsPlayable: &unplayable}, "Mix"); ok {
		t.Fatal("an unplayable video should be dropped")
	}

	if _, ok := videoToSong(playlistVideoRenderer{}, "Mix"); ok {
		t.Fatal("a video without id should be dropped")
	}

	var deleted playlistVideoRenderer
	fixture := `{"videoId":"del12345678","title":{"runs":[{"text":"[Deleted video]"}]}}`
	if err := json.Unmarshal([]byte(fixture), &deleted); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := videoToSong(deleted, "Mix"); ok {
		t.Fatal("a deleted video should be dropped")
	}
}

func TestContinuationParsing(t *testing.T) {
	var res playlistContinuationResponse
	if err := json.Unmarshal([]byte(continuationFixture), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	list := res.ContinuationContents.PlaylistVideoListContinuation
	songs := extractSongs(list, "Some Playlist")
	if len(songs) != 1 {
		t.Fatalf("songs = %d, want 1", len(songs))
	}
	if songs[0].Name != "Song Two" || songs[0].Duration != 180 {
		t.Fatalf("song = %+v", songs[0])
	}
	if songs[0].Album.Title != "Some Playlist" {
		t.Fatalf("album = %q", songs[0].Album.Title)
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
	var res searchResponse
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
	if p.Cover != "https://i.ytimg.com/vi/x/hqdefault.jpg" {
		t.Fatalf("cover = %q", p.Cover)
	}
}

func TestPlaylistIdFromQuery(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/playlist?list=PLabc1234567":       "PLabc1234567",
		"https://www.youtube.com/watch?v=x&list=PLabc1234567&t=1s": "PLabc1234567",
		"PLabc1234567": "PLabc1234567",
		"dQw4w9WgXcQ":  "",
		"peso pluma":   "",
		"PLshort":      "",
	}
	for in, want := range cases {
		if got := playlistIdFromQuery(in); got != want {
			t.Fatalf("playlistIdFromQuery(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestVideoIdFromQuery(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ":                   "dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ&list=PLabc1234567": "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ":                                  "dQw4w9WgXcQ",
		"https://www.youtube.com/shorts/dQw4w9WgXcQ":                    "dQw4w9WgXcQ",
		"dQw4w9WgXcQ": "dQw4w9WgXcQ",
		"https://www.youtube.com/playlist?list=PLabc1234567": "",
		"PLabc1234567": "",
		"peso pluma":   "",
	}
	for in, want := range cases {
		got, ok := videoIdFromQuery(in)
		if want == "" {
			if ok {
				t.Fatalf("videoIdFromQuery(%q) = %q, want none", in, got)
			}
			continue
		}
		if !ok || got != want {
			t.Fatalf("videoIdFromQuery(%q) = %q/%v, want %q", in, got, ok, want)
		}
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

// The web client, the only one that opens a private playlist, nests the list
// under a tab and moves it between releases: the walk has to find it anyway.
const webBrowseFixture = `{
  "contents": {
    "twoColumnBrowseResultsRenderer": {
      "tabs": [{
        "tabRenderer": {
          "content": {
            "sectionListRenderer": {
              "contents": [{
                "itemSectionRenderer": {
                  "contents": [{
                    "playlistVideoListRenderer": {
                      "contents": [
                        { "playlistVideoRenderer": {
                            "videoId": "privateVid1",
                            "title": { "runs": [{ "text": "Private Song" }] },
                            "shortBylineText": { "runs": [{ "text": "Artist" }] },
                            "lengthSeconds": "200"
                        }},
                        { "continuationItemRenderer": {
                            "continuationEndpoint": {
                              "continuationCommand": { "token": "page-two" }
                            }
                        }}
                      ]
                    }
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

func TestWebShapeFindsTheList(t *testing.T) {
	list, ok := playlistList([]byte(webBrowseFixture))
	if !ok {
		t.Fatal("the web shape was not found")
	}
	songs := extractSongs(list, "test")
	if len(songs) != 1 || songs[0].Id != "youtube:privateVid1" {
		t.Fatalf("songs = %+v, want the private one", songs)
	}
	if got := playlistNext([]byte(webBrowseFixture)); got != "page-two" {
		t.Fatalf("next = %q, want the list continuation", got)
	}
}

// The known shape keeps being read by the typed path, so the walk does not
// change what already worked.
func TestKnownShapeStillParses(t *testing.T) {
	var res playlistBrowseResponse
	if err := json.Unmarshal([]byte(browseFixture), &res); err != nil {
		t.Fatal(err)
	}
	if _, ok := res.firstList(); !ok {
		t.Fatal("the known shape stopped parsing")
	}
	if _, ok := playlistList([]byte(browseFixture)); !ok {
		t.Fatal("the walk does not see the known shape")
	}
}

// A private playlist answers 200 with an empty page to the anonymous request, so
// the read has to fall through to the account instead of stopping there.
func TestPickPlaylistPrefersTheReadThatHasTheVideos(t *testing.T) {
	empty := []byte(`{"contents":{"singleColumnBrowseResultsRenderer":{"tabs":[]}}}`)

	got, err := pickPlaylist(empty, nil, []byte(webBrowseFixture), nil, true)
	if err != nil || !hasPlaylistPage(got) {
		t.Fatalf("pick = %v/%v, want the signed page", len(got), err)
	}

	// The readable anonymous page is kept when asking the account fails.
	got, err = pickPlaylist([]byte(browseFixture), nil, nil, errors.New("account unreachable"), true)
	if err != nil || string(got) != browseFixture {
		t.Fatalf("pick = %v/%v, want the anonymous page", len(got), err)
	}

	// Nothing readable on either side, with a full session: the caller reports it
	// as not found; with a partial one, the session is the thing to fix.
	if _, err := pickPlaylist(empty, nil, empty, nil, true); err != nil {
		t.Fatalf("pick = %v, want the not-found path left to the caller", err)
	}
	if _, err := pickPlaylist(empty, nil, empty, nil, false); !errors.Is(err, errSession) {
		t.Fatalf("pick = %v, want the session error", err)
	}

	// YouTube refusing the account is a session problem, not a missing playlist.
	if _, err := pickPlaylist(empty, nil, nil, &apiError{Code: 401}, true); !errors.Is(err, errSession) {
		t.Fatalf("pick = %v, want the session error on 401", err)
	}
}

// The continuation shape carries the list under its own key.
func TestContinuationShapeIsFound(t *testing.T) {
	data := []byte(`{"continuationContents":{"playlistVideoListContinuation":{
		"contents":[{"playlistVideoRenderer":{"videoId":"nextVid","title":{"runs":[{"text":"Next"}]}}}],
		"continuations":[{"nextContinuationData":{"continuation":"page-three"}}]}}}`)

	list, next, ok := playlistPage(data)
	if !ok || len(extractSongs(list, "x")) != 1 || next != "page-three" {
		t.Fatalf("page = %+v/%q/%v, want the continued videos", list, next, ok)
	}
}
