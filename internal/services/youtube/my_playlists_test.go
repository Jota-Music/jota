package youtube

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Jota-Music/jota/internal/music"
)

func TestExtractMyPlaylists(t *testing.T) {
	cases := []struct {
		name  string
		file  string
		id    string
		title string
		owner string
	}{
		{
			name:  "lockupViewModel",
			file:  "testdata/me-playlists-lockup.json",
			id:    "youtube:PLBTanuC8SLeZUH4mYXFvRbDfxTMKvNLHJ",
			title: "Lofi Hip Hop 2026 - Lofi Music 2026 Playlist - Lo fi Playlist",
			owner: "Hitsify",
		},
		{
			name:  "playlistRenderer",
			file:  "testdata/me-playlists-renderer.json",
			id:    "youtube:PLmxm7yELedw-19LktYIv1OtVgisjLXo90",
			title: "Lofi House Mix by Kiffen Beats",
			owner: "Zlone star",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data, err := os.ReadFile(c.file)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			got := extractMyPlaylists(data)
			if len(got) != 2 {
				t.Fatalf("expected 2 playlists, got %d: %+v", len(got), got)
			}
			var want *music.PlaylistSummary
			for i := range got {
				if got[i].Id == c.id {
					want = &got[i]
				}
			}
			if want == nil {
				t.Fatalf("missing playlist %s: %+v", c.id, got)
			}
			if want.Name != c.title {
				t.Fatalf("name mismatch: %q", want.Name)
			}
			if want.Subtitle != c.owner {
				t.Fatalf("owner mismatch: %q", want.Subtitle)
			}
			if !strings.HasPrefix(want.Cover, "https://") {
				t.Fatalf("cover mismatch: %q", want.Cover)
			}
		})
	}
}

func TestGetMyYouTubePlaylistsDoesNotCacheEmpty(t *testing.T) {
	resetAuth(t)
	if err := SetCookies("SAPISID=sapisid123; __Secure-3PSID=sid3p"); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}
	_ = playlistBucket.Delete(myPlaylistsKey)
	t.Cleanup(func() { _ = playlistBucket.Delete(myPlaylistsKey) })

	calls := 0
	old := myPlaylistsFn
	myPlaylistsFn = func() ([]music.PlaylistSummary, error) {
		calls++
		if calls == 1 {
			return nil, errNoPlaylists
		}
		return []music.PlaylistSummary{{Id: "youtube:PL1234567890abcdefgh", Name: "Mix"}}, nil
	}
	t.Cleanup(func() { myPlaylistsFn = old })

	first, err := (&Service{}).GetMyYouTubePlaylists()
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if len(first) != 0 {
		t.Fatalf("expected an empty list, got %+v", first)
	}

	second, err := (&Service{}).GetMyYouTubePlaylists()
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if calls != 2 {
		t.Fatalf("empty result was cached: fetch called %d times, want 2", calls)
	}
	if len(second) != 1 {
		t.Fatalf("expected the fetched playlist, got %+v", second)
	}
}

func TestGetMyYouTubePlaylistsRequiresAuth(t *testing.T) {
	// Ensure no cached state leaks; just check unauthenticated returns error.
	// (HasAuth depends on env/kv; in CI it's false.)
	if HasAuth() {
		t.Skip("account cookies present; skipping unauth check")
	}
	_, err := (&Service{}).GetMyYouTubePlaylists()
	if err == nil {
		t.Fatal("expected error when not signed in")
	}
}

func TestLiveMyYouTubePlaylists(t *testing.T) {
	if os.Getenv("YOUTUBE_COOKIES") == "" {
		t.Skip("YOUTUBE_COOKIES not set")
	}
	if err := SetCookies(os.Getenv("YOUTUBE_COOKIES")); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}
	t.Cleanup(func() { _ = ClearCookies() })

	got, err := (&Service{}).GetMyYouTubePlaylists()
	if err != nil {
		t.Fatalf("GetMyYouTubePlaylists: %v", err)
	}
	t.Logf("got %d playlists", len(got))
	for _, p := range got {
		t.Logf("  %s | %s | %s", p.Id, p.Name, p.Cover)
	}
	// Dump raw response to a file for fixture capture.
	data, _ := json.MarshalIndent(got, "", "  ")
	_ = os.WriteFile("/tmp/opencode/my_playlists.json", data, 0644)
}

// myPlaylistsPage is the shape the sign-in window hands over: a /feed/playlists
// page whose ytInitialData carries the account's playlists as lockupViewModel
// entries.
const myPlaylistsPage = `<!doctype html><html><head><script nonce="abc">var ytInitialData = {
  "contents": {
    "twoColumnBrowseResultsRenderer": {
      "tabs": [
        {
          "tabRenderer": {
            "content": {
              "sectionListRenderer": {
                "contents": [
                  {
                    "itemSectionRenderer": {
                      "contents": [
                        {
                          "lockupViewModel": {
                            "contentId": "PLpage1234567890ab",
                            "contentType": "LOCKUP_CONTENT_TYPE_PLAYLIST",
                            "contentImage": {
                              "collectionThumbnailViewModel": {
                                "primaryThumbnail": {
                                  "thumbnailViewModel": {
                                    "image": {
                                      "sources": [
                                        {"url": "https://i.ytimg.com/vi/GgVcgbtHY9k/hq720.jpg", "width": 720}
                                      ]
                                    }
                                  }
                                }
                              }
                            },
                            "metadata": {
                              "lockupMetadataViewModel": {
                                "title": {"content": "Page Mix"},
                                "metadata": {
                                  "contentMetadataViewModel": {
                                    "metadataRows": [
                                      {"metadataParts": [{"text": {"content": "Jota"}}]}
                                    ]
                                  }
                                }
                              }
                            }
                          }
                        }
                      ]
                    }
                  }
                ]
              }
            }
          }
        }
      ]
    }
  }
};</script></head><body></body></html>`

func TestExtractMyPlaylistsFromPage(t *testing.T) {
	got := extractMyPlaylistsFromPage([]byte(myPlaylistsPage))
	if len(got) != 1 {
		t.Fatalf("expected 1 playlist, got %d: %+v", len(got), got)
	}
	if got[0].Id != "youtube:PLpage1234567890ab" || got[0].Name != "Page Mix" || got[0].Subtitle != "Jota" {
		t.Fatalf("unexpected playlist: %+v", got[0])
	}
	if !strings.HasPrefix(got[0].Cover, "https://i.ytimg.com/") {
		t.Fatalf("cover: %q", got[0].Cover)
	}
}

func TestGetMyYouTubePlaylistsUsesImportedPage(t *testing.T) {
	resetAuth(t)
	// The sign-in window can only hand over the signing cookies, never the
	// HttpOnly session ones.
	if err := SetCookies("SID=abc; SAPISID=sapisid123"); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}
	_ = playlistBucket.Delete(myPlaylistsKey)
	_ = playlistBucket.Delete(myPlaylistsSnapshotKey)
	t.Cleanup(func() { _ = playlistBucket.Delete(myPlaylistsSnapshotKey) })

	if n, err := ImportMyPlaylists([]byte(myPlaylistsPage)); err != nil || n != 1 {
		t.Fatalf("ImportMyPlaylists = %d, %v", n, err)
	}

	called := false
	old := myPlaylistsFn
	myPlaylistsFn = func() ([]music.PlaylistSummary, error) {
		called = true
		return nil, errSession
	}
	t.Cleanup(func() { myPlaylistsFn = old })

	got, err := (&Service{}).GetMyYouTubePlaylists()
	if err != nil {
		t.Fatalf("GetMyYouTubePlaylists: %v", err)
	}
	if called {
		t.Fatal("innertube should not be asked when the session is the partial one")
	}
	if len(got) != 1 || got[0].Name != "Page Mix" {
		t.Fatalf("unexpected playlists: %+v", got)
	}
}

func TestGetMyYouTubePlaylistsReportsRejectedSession(t *testing.T) {
	resetAuth(t)
	if err := SetCookies("SID=abc; SAPISID=sapisid123"); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}
	_ = playlistBucket.Delete(myPlaylistsSnapshotKey)

	_, err := (&Service{}).GetMyYouTubePlaylists()
	if !errors.Is(err, errSession) {
		t.Fatalf("expected errSession, got %v", err)
	}
}

// A guide response carries the library link the web app uses, with the params
// the hardcoded browse id lacks.
const guideFixture = `{
  "items": [{
    "guideSectionRenderer": {
      "items": [{
        "guideEntryRenderer": {
          "formattedTitle": { "simpleText": "Playlists" },
          "navigationEndpoint": {
            "browseEndpoint": { "browseId": "FEplaylists", "params": "EgdsaWJyYXJ5" }
          }
        }
      }]
    }
  }]
}`

func TestFindPlaylistsTarget(t *testing.T) {
	id, params, ok := findPlaylistsTarget([]byte(guideFixture))
	if !ok || id != "FEplaylists" || params != "EgdsaWJyYXJ5" {
		t.Fatalf("target = %q/%q/%v, want the guide's link", id, params, ok)
	}

	if _, _, ok := findPlaylistsTarget([]byte(`{"items":[]}`)); ok {
		t.Fatal("a guide without the entry answered a target")
	}
}

// usePage serves page at myPlaylistsPageURL for the duration of a test.
func usePage(t *testing.T, page []byte) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(page)
	}))
	old := myPlaylistsPageURL
	myPlaylistsPageURL = server.URL
	t.Cleanup(func() {
		myPlaylistsPageURL = old
		server.Close()
	})
}

func TestGetMyYouTubePlaylistsReadsPage(t *testing.T) {
	resetAuth(t)
	if err := SetCookies("SID=abc; SAPISID=sapisid123; __Secure-3PSID=sid3p"); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}
	usePage(t, []byte(myPlaylistsPage))
	_ = playlistBucket.Delete(myPlaylistsKey)
	_ = playlistBucket.Delete(myPlaylistsSnapshotKey)
	t.Cleanup(func() {
		_ = playlistBucket.Delete(myPlaylistsKey)
		_ = playlistBucket.Delete(myPlaylistsSnapshotKey)
	})

	got, err := (&Service{}).GetMyYouTubePlaylists()
	if err != nil {
		t.Fatalf("GetMyYouTubePlaylists: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Page Mix" {
		t.Fatalf("unexpected playlists: %+v", got)
	}
}

func TestGetMyYouTubePlaylistsFallsBackToSnapshot(t *testing.T) {
	resetAuth(t)
	if err := SetCookies("SID=abc; SAPISID=sapisid123; __Secure-3PSID=sid3p"); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}
	_ = playlistBucket.Delete(myPlaylistsKey)
	t.Cleanup(func() { _ = playlistBucket.Delete(myPlaylistsKey) })
	if n, err := ImportMyPlaylists([]byte(myPlaylistsPage)); err != nil || n != 1 {
		t.Fatalf("ImportMyPlaylists = %d, %v", n, err)
	}
	t.Cleanup(func() { _ = playlistBucket.Delete(myPlaylistsSnapshotKey) })

	old := myPlaylistsFn
	myPlaylistsFn = func() ([]music.PlaylistSummary, error) {
		return nil, errors.New("dial tcp 142.250.200.14:443: connect: network is unreachable")
	}
	t.Cleanup(func() { myPlaylistsFn = old })

	got, err := (&Service{}).GetMyYouTubePlaylists()
	if err != nil {
		t.Fatalf("GetMyYouTubePlaylists: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Page Mix" {
		t.Fatalf("unexpected playlists: %+v", got)
	}
}

func TestGetMyYouTubePlaylistsServesSnapshotWithoutAuth(t *testing.T) {
	resetAuth(t)
	_ = playlistBucket.Delete(myPlaylistsKey)
	if n, err := ImportMyPlaylists([]byte(myPlaylistsPage)); err != nil || n != 1 {
		t.Fatalf("ImportMyPlaylists = %d, %v", n, err)
	}
	t.Cleanup(func() { _ = playlistBucket.Delete(myPlaylistsSnapshotKey) })

	got, err := (&Service{}).GetMyYouTubePlaylists()
	if err != nil {
		t.Fatalf("GetMyYouTubePlaylists: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Page Mix" {
		t.Fatalf("unexpected playlists: %+v", got)
	}
}

func TestSetCookiesDropsCachedPlaylists(t *testing.T) {
	resetAuth(t)
	if err := SetCookies("SID=abc; SAPISID=sapisid123; __Secure-3PSID=sid3p"); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}
	_ = playlistBucket.Delete(myPlaylistsKey)
	_ = playlistBucket.Delete(myPlaylistsSnapshotKey)
	t.Cleanup(func() {
		_ = playlistBucket.Delete(myPlaylistsKey)
		_ = playlistBucket.Delete(myPlaylistsSnapshotKey)
	})

	calls := 0
	old := myPlaylistsFn
	myPlaylistsFn = func() ([]music.PlaylistSummary, error) {
		calls++
		return []music.PlaylistSummary{{Id: "youtube:PL1234567890abcdefgh", Name: "Mix"}}, nil
	}
	t.Cleanup(func() { myPlaylistsFn = old })

	if _, err := (&Service{}).GetMyYouTubePlaylists(); err != nil {
		t.Fatalf("first call: %v", err)
	}

	if err := SetCookies("SID=xyz; SAPISID=sapisid456; __Secure-3PSID=sid3p2"); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}
	if _, err := (&Service{}).GetMyYouTubePlaylists(); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if calls != 2 {
		t.Fatalf("the previous account's list survived SetCookies: fetch called %d times, want 2", calls)
	}
}
