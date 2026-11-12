package youtube

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/Jota-Music/jota/internal/cache"
	"github.com/Jota-Music/jota/internal/music"
)

func seedYoutubeId(song, id string) {
	st := readSession()
	if st.Links == nil {
		st.Links = map[string]string{}
	}
	if id == "" {
		delete(st.Links, song)
	} else {
		st.Links[song] = id
	}
	_ = writeSession(st)
}

func TestSetYoutubeIdInvalidatesAudioCache(t *testing.T) {
	const song = "test-set-youtube-id"
	future := time.Now().Add(time.Hour).Unix()
	keepAudio("old_video01", music.Audio{Url: "https://example.com/a", VideoID: "old_video01", ExpireAt: future}, time.Hour)
	seedYoutubeId(song, "old_video01")

	svc := NewService()
	if err := svc.SetYoutubeId(song, "new_video01"); err != nil {
		t.Fatalf("SetYoutubeId: %v", err)
	}
	if got := cachedAudio("old_video01"); got != nil {
		t.Fatalf("the stream of the old video was not invalidated: %+v", got)
	}
}

// A stream resolved by this run was probed by the resolve, so playing it again
// must not pay a HEAD to googlevideo. A stream the process did not resolve is
// still probed, unless it was resolved so recently that its URL cannot have been
// revoked yet: the resolve already probed it, and the signature has hours left.
func TestAStreamResolvedThisRunIsNotProbedAgain(t *testing.T) {
	const video = "fresh_video01"
	keepAudio(video, music.Audio{
		Url:      "http://127.0.0.1:1/gone",
		VideoID:  video,
		ExpireAt: time.Now().Add(time.Hour).Unix(),
	}, time.Hour)

	// Just resolved, in this run: no HEAD, even though the URL is dead.
	if got := cachedAudio(video); got == nil {
		t.Fatal("a stream this run resolved was probed again and dropped")
	}

	// Same entry, but the process did not resolve it: the disk says it was
	// written a moment ago, so it is still trusted without a HEAD.
	fresh.Delete(video)
	recentSaved := time.Now().UnixMilli()
	raw := fmt.Sprintf(`{"name":%q,"saved":%d,"value":{"url":"http://127.0.0.1:1/gone","videoId":%q,"expireAt":%d}}`,
		streamName(video), recentSaved, video, time.Now().Add(time.Hour).Unix())
	if err := cache.Write(streamName(video), []byte(raw)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := cachedAudio(video); got == nil {
		t.Fatal("a recently resolved stream was probed again and dropped")
	}

	// An old entry is not trusted blindly: it is probed, and a dead URL drops it.
	oldSaved := time.Now().Add(-time.Hour).UnixMilli()
	raw = fmt.Sprintf(`{"name":%q,"saved":%d,"value":{"url":"http://127.0.0.1:1/gone","videoId":%q,"expireAt":%d}}`,
		streamName(video), oldSaved, video, time.Now().Add(time.Hour).Unix())
	if err := cache.Write(streamName(video), []byte(raw)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := cachedAudio(video); got != nil {
		t.Fatal("an old stream was played without probing it")
	}
}

// The resolved stream lives in a file of its own, so a play rewrites one entry
// and not the whole YouTube cache. The session file keeps the tokens alone.
func TestTheStreamLivesInAFileOfItsOwn(t *testing.T) {
	const video = "own_file_video1"
	keepAudio(video, music.Audio{Url: "https://example.com/a", VideoID: video, ExpireAt: time.Now().Add(time.Hour).Unix()}, time.Hour)
	tokens := &sessionCache{}
	tokens.rememberClient("IOS")

	if _, ok := cache.Get[music.Audio](streamName(video)); !ok {
		t.Fatal("the stream is not in its own file")
	}

	data, ok := cache.ReadRaw(sessionName)
	if !ok {
		t.Fatal("the session file is not there")
	}
	var stored map[string]json.RawMessage
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("the session file will not decode: %v", err)
	}
	if _, ok := stored["streams"]; ok {
		t.Fatal("the session file still carries the streams")
	}
}

// A device that keeps the streams in the session file, the old layout, keeps
// its tokens and drops what the streams left on the next write.
func TestTheOldSessionFileKeepsTheTokensAndDropsTheStreams(t *testing.T) {
	old := `{"streams":{"a_video01":{"url":"u"}},"visitor":{"value":"token-1","at":1},"client":"IOS"}`
	if err := cache.Write(sessionName, []byte(old)); err != nil {
		t.Fatalf("write: %v", err)
	}

	fresh := &sessionCache{}
	if got := fresh.client(); got != "IOS" {
		t.Fatalf("client = %q, want %q", got, "IOS")
	}
	fresh.rememberClient("ANDROID")

	data, ok := cache.ReadRaw(sessionName)
	if !ok {
		t.Fatal("the session file is not there")
	}
	var stored map[string]json.RawMessage
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("the session file will not decode: %v", err)
	}
	if _, ok := stored["streams"]; ok {
		t.Fatal("the streams are still in the session file")
	}
	if _, ok := stored["visitor"]; !ok {
		t.Fatal("the visitor was lost")
	}
}

func TestSetYoutubeIdStoresOnlyTheId(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ": "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ":                "dQw4w9WgXcQ",
		"https://www.youtube.com/shorts/dQw4w9WgXcQ":  "dQw4w9WgXcQ",
		"youtube:dQw4w9WgXcQ":                         "dQw4w9WgXcQ",
	}

	for input, want := range cases {
		const song = "test-set-youtube-id-url"
		seedYoutubeId(song, "")

		if err := NewService().SetYoutubeId(song, input); err != nil {
			t.Fatalf("SetYoutubeId(%q): %v", input, err)
		}
		if got := NewService().YoutubeId(song); got != want {
			t.Fatalf("YoutubeId = %q, want %q", got, want)
		}
	}
}

func TestSetYoutubeIdRejectsInvalid(t *testing.T) {
	if err := NewService().SetYoutubeId("test-invalid-youtube", "not a video"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestYoutubeIdLookup(t *testing.T) {
	const song = "test-youtube-id-lookup"
	seedYoutubeId(song, "video1")

	svc := NewService()
	if got := svc.YoutubeId(song); got != "video1" {
		t.Fatalf("YoutubeId = %q, want %q", got, "video1")
	}
	if got := svc.YoutubeId("unknown-song"); got != "" {
		t.Fatalf("YoutubeId = %q, want empty", got)
	}
}

// Find must answer from the cached id without searching: a plain text query
// with no stub responses would fail the search path.
func TestFindUsesCachedId(t *testing.T) {
	const song = "test-find-cached"
	seedYoutubeId(song, "cached_video1")
	stubSearch(t, map[string]searchResponse{})

	id, err := NewService().Find(music.Song{Id: song, Name: "Rambo"})
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if id != "cached_video1" {
		t.Fatalf("Find = %q, want the cached id", id)
	}
}

func TestFindSearchesAndRecordsId(t *testing.T) {
	const song = "test-find-search"
	seedYoutubeId(song, "")

	want := music.Song{
		Id:       song,
		Name:     "Rambo",
		Artists:  []music.Artist{{Name: "Orquesta Mondragón"}},
		Album:    music.Album{Title: "Ellos las prefieren gordas"},
		Duration: 182,
	}
	stubSearch(t, map[string]searchResponse{
		audioQuery(want): searchVideos(
			Video{ID: "ohV8ZwFXQN4", Title: "Rambo", Author: "Orquesta Mondragón - Topic", Duration: 183},
		),
	})

	svc := NewService()
	id, err := svc.Find(want)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if id != "ohV8ZwFXQN4" {
		t.Fatalf("Find = %q, want the searched video", id)
	}
	if got := svc.YoutubeId(song); got != id {
		t.Fatalf("YoutubeId = %q, want the searched id recorded", got)
	}
}

func TestGetAudioURL(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: reaches YouTube")
	}
	url, client, err := audioURL("E9s9BNZFQLA")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if url == "" || len(url) < 50 {
		t.Fatalf("invalid URL: %s", url)
	}
	if client.Name == "" {
		t.Fatalf("expected client name, got empty")
	}
}

// The refresh walks the cache, so the entries it must revisit have to be
// where the walk looks: the account's playlist list sits next to the folders,
// not inside one.
func TestRefreshVisitsTheAccountPlaylistList(t *testing.T) {
	if err := cache.Put(myPlaylistsKey, []music.PlaylistSummary{}); err != nil {
		t.Fatalf("put: %v", err)
	}

	seen := false
	for _, dir := range refreshDirs {
		err := cache.Walk(dir, func(name string) error {
			if name == myPlaylistsKey {
				seen = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %q: %v", dir, err)
		}
	}
	if !seen {
		t.Fatalf("the refresh does not visit %q", myPlaylistsKey)
	}
}

// The folder of playlists holds two kinds of entity, and the id tells them
// apart: a channel id starts with UC and a playlist id never does.
func TestThePlaylistFolderTellsBothKindsApart(t *testing.T) {
	const channel = "UC" + "abcdefghijklmnopqrstuv"
	const playlist = "PLabcdefghijklmnopqrstuv"

	if !channelIDPattern.MatchString(channel) {
		t.Fatal("the channel id is not recognized as one")
	}
	if channelIDPattern.MatchString(playlist) {
		t.Fatal("the playlist id is taken for a channel id")
	}
}
