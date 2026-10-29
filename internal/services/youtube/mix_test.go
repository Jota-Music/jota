package youtube

import "testing"

const mixFixture = `{
  "contents": {
    "twoColumnWatchNextResults": {
      "playlist": {
        "playlist": {
          "title": "Mix - Camila",
          "longBylineText": { "runs": [{ "text": "YouTube Music" }] },
          "contents": [
            { "playlistPanelVideoRenderer": {
                "videoId": "dQw4w9WgXcQ",
                "title": { "simpleText": "Song One" },
                "longBylineText": { "runs": [{ "text": "Artist One" }] },
                "lengthText": { "simpleText": "1:02:03" },
                "thumbnail": { "thumbnails": [{ "url": "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg" }] }
            }},
            { "playlistPanelVideoRenderer": {
                "videoId": "aaaaaaaaaaa",
                "title": { "simpleText": "Song Two" },
                "lengthText": { "simpleText": "3:11" },
                "thumbnail": { "thumbnails": [{ "url": "https://i.ytimg.com/vi/aaaaaaaaaaa/hqdefault.jpg" }] }
            }},
            { "playlistPanelVideoRenderer": { "videoId": "", "title": { "simpleText": "No Id" } } },
            { "playlistPanelVideoRenderer": {
                "videoId": "bbbbbbbbbbb",
                "title": { "simpleText": "[Deleted video]" },
                "thumbnail": { "thumbnails": [{ "url": "https://i.ytimg.com/vi/bbbbbbbbbbb/hqdefault.jpg" }] }
            }},
            { "playlistVideoRenderer": { "videoId": "ccccccccccc" } }
          ]
        }
      }
    }
  }
}`

func TestMixSeedIsTheVideoTheMixIsBuiltAround(t *testing.T) {
	cases := map[string]string{
		"RDg4LgByUx-cU":       "g4LgByUx-cU",
		"RDdQw4w9WgXcQ":       "dQw4w9WgXcQ",
		"PL1234567890abcdef":  "",
		"RD":                  "",
		"RDshort":             "",
		"RDCLAK5uy_kABCDEFGH": "",
	}
	for in, want := range cases {
		if got := mixSeed(in); got != want {
			t.Fatalf("mixSeed(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMixParsesThePlaylistPanel(t *testing.T) {
	playlist, err := parseMix([]byte(mixFixture))
	if err != nil {
		t.Fatalf("parseMix: %v", err)
	}

	if playlist.Name != "Mix - Camila" {
		t.Fatalf("name = %q", playlist.Name)
	}
	if playlist.Owner != "YouTube Music" {
		t.Fatalf("owner = %q", playlist.Owner)
	}
	if playlist.Cover != "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg" {
		t.Fatalf("cover = %q", playlist.Cover)
	}
	if len(playlist.Songs) != 2 {
		t.Fatalf("songs = %d, want 2", len(playlist.Songs))
	}

	first := playlist.Songs[0]
	if first.Id != "youtube:dQw4w9WgXcQ" || first.YoutubeId != "dQw4w9WgXcQ" {
		t.Fatalf("id = %q/%q", first.Id, first.YoutubeId)
	}
	if first.Duration != 3723 {
		t.Fatalf("duration = %d", first.Duration)
	}
	if first.Album.Title != "Mix - Camila" {
		t.Fatalf("album = %q", first.Album.Title)
	}
	if len(first.Artists) != 1 || first.Artists[0].Name != "Artist One" {
		t.Fatalf("artists = %+v", first.Artists)
	}
	if playlist.Songs[1].Duration != 191 {
		t.Fatalf("second duration = %d", playlist.Songs[1].Duration)
	}
}

func TestMixWithoutVideosIsAnError(t *testing.T) {
	if _, err := parseMix([]byte(`{"contents":{}}`)); err == nil {
		t.Fatal("an empty mix must not parse")
	}
}
