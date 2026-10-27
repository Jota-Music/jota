package youtube

import (
	"strings"
	"testing"
)

// webPlaylistJSON is the shape the web client answers an own or private
// playlist with: the title in a modern page header and the videos as
// lockupViewModel items, next to entries that are playlists themselves.
const webPlaylistJSON = `{
  "header": {"pageHeaderRenderer": {"pageTitle": "Mi lista"}},
  "contents": {"twoColumnBrowseResultsRenderer": {"tabs": [{"tabRenderer": {"content": {"sectionListRenderer": {"contents": [
    {"itemSectionRenderer": {"contents": [
      {"lockupViewModel": {
        "contentId": "bAlxY-bUv84",
        "contentType": "LOCKUP_CONTENT_TYPE_VIDEO",
        "contentImage": {"thumbnailViewModel": {
          "image": {"sources": [
            {"url": "https://i.ytimg.com/vi/bAlxY-bUv84/hqdefault.jpg", "width": 336},
            {"url": "https://i.ytimg.com/vi/bAlxY-bUv84/hq720.jpg", "width": 1280}
          ]},
          "overlays": [{"thumbnailBottomOverlayViewModel": {"badges": [
            {"thumbnailBadgeViewModel": {"text": "1:02:03"}}
          ]}}]
        }},
        "metadata": {"lockupMetadataViewModel": {
          "title": {"content": "Bolero instrumental"},
          "metadata": {"contentMetadataViewModel": {"metadataRows": [
            {"metadataParts": [{"text": {"content": "prod.dorado"}, "avatarStack": {"avatarStackViewModel": {}}}]}
          ]}}
        }}
      }},
      {"lockupViewModel": {
        "contentId": "PLpage1234567890ab",
        "contentType": "LOCKUP_CONTENT_TYPE_PLAYLIST",
        "metadata": {"lockupMetadataViewModel": {"title": {"content": "Otra lista"}}}
      }}
    ]}}
  ]}}}}]}}
}`

func TestWebPlaylistPage(t *testing.T) {
	songs, title := webPlaylistPage([]byte(webPlaylistJSON))
	if title != "Mi lista" {
		t.Fatalf("title = %q", title)
	}
	if len(songs) != 1 {
		t.Fatalf("expected one song, got %d: %+v", len(songs), songs)
	}
	song := songs[0]
	if song.Id != "youtube:bAlxY-bUv84" || song.YoutubeId != "bAlxY-bUv84" {
		t.Fatalf("id = %q / %q", song.Id, song.YoutubeId)
	}
	if song.Name != "Bolero instrumental" {
		t.Fatalf("name = %q", song.Name)
	}
	if song.Duration != 3723 {
		t.Fatalf("duration = %d", song.Duration)
	}
	if song.Album.Title != "Mi lista" {
		t.Fatalf("album = %q", song.Album.Title)
	}
	if len(song.Artists) != 1 || song.Artists[0].Name != "prod.dorado" {
		t.Fatalf("artists = %+v", song.Artists)
	}
	if len(song.Album.Covers) != 1 || !strings.HasSuffix(song.Album.Covers[0], "hqdefault.jpg") {
		t.Fatalf("cover = %+v", song.Album.Covers)
	}
}

func TestPageTitle(t *testing.T) {
	if got := pageTitle([]byte(webPlaylistJSON)); got != "Mi lista" {
		t.Fatalf("pageTitle = %q", got)
	}
	if got := pageTitle([]byte(`{"header":{}}`)); got != "" {
		t.Fatalf("pageTitle = %q, want empty", got)
	}
}

func TestClockSeconds(t *testing.T) {
	cases := map[string]int{"0:14": 14, "7:52": 472, "1:02:03": 3723, "": 0, "lol": 0, "1:2:3:4": 0}
	for text, want := range cases {
		if got := clockSeconds(text); got != want {
			t.Fatalf("clockSeconds(%q) = %d, want %d", text, got, want)
		}
	}
}
