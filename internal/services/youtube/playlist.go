package youtube

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

var playlistBucket = kv.UseBucket("youtube-playlists")

const browseEndpoint = "https://www.youtube.com/youtubei/v1/browse"

func normalizePlaylistId(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, music.YouTubePrefix)
	id = strings.TrimPrefix(id, "VL")
	return id
}

func postBrowse(payload map[string]any) ([]byte, error) {
	return postBrowseWithClient(preferredClient, payload)
}

// apiError is an innertube response that carried an error object instead of
// data. The code is kept so callers can tell an unauthenticated session from a
// request that was simply malformed.
type apiError struct {
	Code    int
	Message string
}

func (e *apiError) Error() string { return "browse error: " + e.Message }

// postBrowseWithClient is the same as postBrowse but lets the caller pick the
// innertube client. The web client is the one YouTube ties to the account
// session, so reading a private playlist needs it; the VISIONOS client is the
// anonymous one.
func postBrowseWithClient(c clientConfig, payload map[string]any) ([]byte, error) {
	return browseResult(retryRequest(c, browseEndpoint, payload, 3))
}

// postBrowseAnonymously browses without the account session, so a session
// YouTube rejects cannot poison the public request that follows it.
func postBrowseAnonymously(c clientConfig, payload map[string]any) ([]byte, error) {
	return browseResult(retryAnonymous(c, browseEndpoint, payload, 3))
}

func browseResult(data []byte, err error) ([]byte, error) {
	if err != nil {
		return nil, fmt.Errorf("browse request failed: %w", err)
	}

	var res struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &res); err == nil && res.Error.Code != 0 {
		return nil, &apiError{Code: res.Error.Code, Message: res.Error.Message}
	}

	return data, nil
}

// playlistPayload builds the browse body for a playlist page or its next page of
// videos, for the given innertube client.
func playlistPayload(c clientConfig, playlistID, continuation string) map[string]any {
	payload := map[string]any{
		"context": map[string]any{"client": clientContext(c)},
	}
	if continuation != "" {
		payload["continuation"] = continuation
	} else {
		payload["browseId"] = "VL" + playlistID
	}
	return payload
}

// signedPlaylistClient is the client a private or own playlist is read with: the
// account session is a web session, so the web client is the only one that both
// carries the cookies and accepts them for a playlist the account owns.
var signedPlaylistClient = webClient()

// browseData reads a playlist, anonymously first: that is the path every public
// playlist takes and the shape this package parses. The account is asked next
// whenever the anonymous answer carries no playlist at all, which is what a
// private one looks like: YouTube answers 200 with an empty page instead of an
// error, so "the request worked" never means "the playlist is readable".
func browseData(playlistID, continuation string) ([]byte, error) {
	anonymous, anonymousErr := postBrowseAnonymously(preferredClient, playlistPayload(preferredClient, playlistID, continuation))
	if anonymousErr == nil && hasPlaylistPage(anonymous) {
		return anonymous, nil
	}
	if !HasAuth() {
		return anonymous, anonymousErr
	}

	signed, signedErr := postBrowseWithClient(signedPlaylistClient, playlistPayload(signedPlaylistClient, playlistID, continuation))
	return pickPlaylist(anonymous, anonymousErr, signed, signedErr, sessionValue(cookies()) != "")
}

// pickPlaylist answers which of the two reads to keep: the signed one when it
// carries the playlist, the anonymous one when the account could not open it.
// A session YouTube refuses is reported as such, because that is what the user
// can act on.
func pickPlaylist(anonymous []byte, anonymousErr error, signed []byte, signedErr error, hasSession bool) ([]byte, error) {
	if signedErr != nil {
		var api *apiError
		if errors.As(signedErr, &api) && api.Code == http.StatusUnauthorized {
			return nil, errSession
		}
		if hasPlaylistPage(anonymous) {
			return anonymous, anonymousErr
		}
		return nil, signedErr
	}
	if hasPlaylistPage(signed) {
		return signed, nil
	}
	// A partial session (the sign-in window cannot read the HttpOnly cookies)
	// signs requests without opening anything: say so instead of "not found".
	if !hasSession {
		return nil, errSession
	}
	if hasPlaylistPage(anonymous) {
		return anonymous, anonymousErr
	}
	return signed, nil
}

// hasPlaylistPage reports whether a browse answer carries a playlist at all. A
// byte scan is enough, and it keeps the public path from parsing the page twice.
func hasPlaylistPage(data []byte) bool {
	return bytes.Contains(data, []byte(`"playlistVideoListRenderer"`)) ||
		bytes.Contains(data, []byte(`"playlistVideoListContinuation"`))
}

// Innertube filters that open a channel's "Playlists" or "Videos" tab, so one
// browse request returns only that tab.
const (
	channelPlaylistsParams = "EglwbGF5bGlzdHPyBgoKCEIGCgIQaCIA"
	channelVideosParams    = "EgZ2aWRlb3PyBgQKAjoA"
)

// channelIDPattern matches a YouTube channel ID (UC + 22 chars).
var channelIDPattern = regexp.MustCompile(`^UC[A-Za-z0-9_-]{22}$`)

// resolveChannelID turns a channel URL, @handle, or bare channel ID into the
// channel ID the browse endpoint expects. Non-ID inputs are resolved through
// innertube and cached.
func resolveChannelID(channel string) (string, error) {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return "", errors.New("empty channel")
	}
	if channelIDPattern.MatchString(channel) {
		return channel, nil
	}

	return kv.Cached(playlistBucket, "channel-id:"+channel, func() (string, error) {
		target := channel
		if !strings.Contains(channel, "://") {
			target = "https://www.youtube.com/" + channel
		}

		payload := map[string]any{
			"url":     target,
			"context": map[string]any{"client": clientContext(preferredClient)},
		}
		data, err := retryRequest(preferredClient, "https://www.youtube.com/youtubei/v1/navigation/resolve_url", payload, 3)
		if err != nil {
			return "", fmt.Errorf("resolve channel failed: %w", err)
		}

		var res resolveURLResponse
		if err := json.Unmarshal(data, &res); err != nil {
			return "", fmt.Errorf("invalid resolve JSON: %w", err)
		}
		id := res.Endpoint.BrowseEndpoint.BrowseID
		if id == "" {
			return "", fmt.Errorf("channel not found: %s", channel)
		}
		return id, nil
	})
}

func channelBrowseData(channelID, params, continuation string) ([]byte, error) {
	payload := map[string]any{
		"context": map[string]any{"client": clientContext(preferredClient)},
	}
	if continuation != "" {
		payload["continuation"] = continuation
	} else {
		payload["browseId"] = channelID
		payload["params"] = params
	}

	return postBrowse(payload)
}

// fetchChannelTab pages through one channel tab, collecting items with extract.
// The channel name comes from the tab header so songs can be attributed to it.
func fetchChannelTab[T any](channelID, params string, extract func(itemSection, string) []T) ([]T, error) {
	data, err := channelBrowseData(channelID, params, "")
	if err != nil {
		return nil, err
	}

	var res channelBrowseResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("invalid channel JSON: %w", err)
	}

	section, ok := res.section()
	if !ok {
		return nil, errors.New("channel tab is empty")
	}

	name := res.Header.C4TabbedHeaderRenderer.Title
	items := extract(section, name)
	continuation := section.next()

	for pages := 1; continuation != ""; pages++ {
		// ponytail: cap the loop so a repeated continuation token can't spin forever
		if pages > 500 {
			return nil, errors.New("channel too large or malformed continuation")
		}

		data, err := channelBrowseData(channelID, params, continuation)
		if err != nil {
			return nil, err
		}

		var cont channelContinuationResponse
		if err := json.Unmarshal(data, &cont); err != nil {
			return nil, fmt.Errorf("invalid channel continuation JSON: %w", err)
		}

		section := cont.ContinuationContents.ItemSectionContinuation
		items = append(items, extract(section, name)...)
		continuation = section.next()
	}

	return items, nil
}

// GetChannelPlaylists returns the playlists owned by a YouTube channel,
// accepting a channel ID, @handle, or channel URL.
func (s *Service) GetChannelPlaylists(channel string) ([]music.PlaylistSummary, error) {
	channelID, err := resolveChannelID(channel)
	if err != nil {
		return nil, err
	}
	return kv.Cached(playlistBucket, "channel:playlists:"+channelID, func() ([]music.PlaylistSummary, error) {
		return fetchChannelTab(channelID, channelPlaylistsParams, func(s itemSection, _ string) []music.PlaylistSummary {
			return s.summaries()
		})
	})
}

// GetChannelVideos returns the videos published by a YouTube channel.
func (s *Service) GetChannelVideos(channel string) ([]music.Song, error) {
	channelID, err := resolveChannelID(channel)
	if err != nil {
		return nil, err
	}
	return kv.Cached(playlistBucket, "channel:videos:"+channelID, func() ([]music.Song, error) {
		return fetchChannelTab(channelID, channelVideosParams, func(s itemSection, name string) []music.Song {
			return s.videos(channelID, name)
		})
	})
}

func (s *Service) RevalidateChannel(channel string) error {
	channelID, err := resolveChannelID(channel)
	if err != nil {
		return err
	}
	_ = playlistBucket.Delete("channel:playlists:" + channelID)
	_ = playlistBucket.Delete("channel:videos:" + channelID)
	return playlistBucket.Delete("channel:info:" + channelID)
}

// GetChannelInfo returns a channel's display name and avatar.
func (s *Service) GetChannelInfo(channel string) (music.ChannelInfo, error) {
	channelID, err := resolveChannelID(channel)
	if err != nil {
		return music.ChannelInfo{}, err
	}
	return kv.Cached(playlistBucket, "channel:info:"+channelID, func() (music.ChannelInfo, error) {
		return channelInfo(channelID)
	})
}

func channelInfo(channelID string) (music.ChannelInfo, error) {
	data, err := channelBrowseData(channelID, channelPlaylistsParams, "")
	if err != nil {
		return music.ChannelInfo{}, err
	}

	var res channelBrowseResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return music.ChannelInfo{}, fmt.Errorf("invalid channel JSON: %w", err)
	}

	header := res.Header.C4TabbedHeaderRenderer
	return music.ChannelInfo{
		Id:     channelID,
		Name:   header.Title,
		Avatar: header.Avatar.url(),
	}, nil
}

func (r channelBrowseResponse) section() (itemSection, bool) {
	for _, tab := range r.Contents.SingleColumnBrowseResultsRenderer.Tabs {
		for _, content := range tab.TabRenderer.Content.SectionListRenderer.Contents {
			if len(content.ItemSectionRenderer.Contents) > 0 {
				return content.ItemSectionRenderer, true
			}
		}
	}
	return itemSection{}, false
}

func (s itemSection) summaries() []music.PlaylistSummary {
	out := make([]music.PlaylistSummary, 0, len(s.Contents))
	for _, item := range s.Contents {
		if summary, ok := item.CompactPlaylistRenderer.summary(); ok {
			out = append(out, summary)
		}
	}
	return out
}

func (s itemSection) videos(channelID, channelName string) []music.Song {
	out := make([]music.Song, 0, len(s.Contents))
	for _, item := range s.Contents {
		v := item.CompactVideoRenderer
		id := strings.TrimSpace(v.VideoID)
		if id == "" {
			continue
		}
		out = append(out, music.Song{
			Id:        music.YouTubePrefix + id,
			Url:       "https://www.youtube.com/watch?v=" + id,
			Name:      v.Title.first(),
			Duration:  v.duration(),
			Share:     music.Share{Id: music.YouTubePrefix + id, Url: "https://www.youtube.com/watch?v=" + id},
			Album:     music.Album{Title: "YouTube", Covers: []string{v.Thumbnail.url()}},
			Artists:   []music.Artist{{Id: channelID, Name: channelName, Source: "youtube"}},
			YoutubeId: id,
		})
	}
	return out
}

func (s itemSection) next() string {
	if len(s.Continuations) == 0 {
		return ""
	}
	return s.Continuations[0].NextContinuationData.Continuation
}

func fetchFullPlaylist(playlistID string) (music.Playlist, error) {
	data, err := browseData(playlistID, "")
	if err != nil {
		return music.Playlist{}, err
	}

	var res playlistBrowseResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return music.Playlist{}, fmt.Errorf("invalid browse JSON: %w", err)
	}

	name, cover, owner, ownerID := "", "", "", ""
	if header, ok := res.headerRenderer(); ok {
		name = header.Title.first()
		cover = header.Banner.HeroPlaylistThumbnailRenderer.Thumbnail.url()
		owner = header.OwnerText.first()
		ownerID = header.OwnerText.browseID()
	}

	list, next, ok := playlistPage(data)
	if !ok {
		// The page a session gets for an own or private playlist keeps its
		// videos in lockupViewModel items and its title in pageHeaderRenderer.
		songs, title := webPlaylistPage(data)
		if name == "" {
			name = title
		}
		// A header means the playlist exists; it may simply hold no video.
		if len(songs) == 0 && name == "" {
			return music.Playlist{}, errors.New("playlist not found or has no videos")
		}
		if cover == "" && len(songs) > 0 && len(songs[0].Album.Covers) > 0 {
			cover = songs[0].Album.Covers[0]
		}
		return music.Playlist{Name: name, Cover: cover, Owner: owner, OwnerId: ownerID, Songs: songs}, nil
	}

	if name == "" {
		// A private or own playlist answers with the modern page header, which
		// keeps its title outside playlistHeaderRenderer.
		name = pageTitle(data)
	}

	songs := extractSongs(list, name)
	continuation := next

	for pages := 1; continuation != ""; pages++ {
		// ponytail: cap the loop so a repeated continuation token can't spin forever
		if pages > 500 {
			return music.Playlist{}, errors.New("playlist too large or malformed continuation")
		}

		data, err := browseData(playlistID, continuation)
		if err != nil {
			return music.Playlist{}, err
		}

		list, next, ok := playlistPage(data)
		if !ok {
			return music.Playlist{}, errors.New("playlist page lost its videos mid-read")
		}
		songs = append(songs, extractSongs(list, name)...)
		continuation = next
	}

	return music.Playlist{
		Name:    name,
		Cover:   cover,
		Owner:   owner,
		OwnerId: ownerID,
		Songs:   songs,
	}, nil
}

func fetchPlaylistSummary(playlistID string) (music.PlaylistSummary, error) {
	data, err := browseData(playlistID, "")
	if err != nil {
		return music.PlaylistSummary{}, err
	}

	var res playlistBrowseResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return music.PlaylistSummary{}, fmt.Errorf("invalid browse JSON: %w", err)
	}

	header, ok := res.headerRenderer()
	if !ok || header.Title.first() == "" {
		// A private or own playlist answers with the modern page header, which
		// keeps its title outside playlistHeaderRenderer.
		header.Title.SimpleText = pageTitle(data)
	}
	if header.Title.first() == "" {
		return music.PlaylistSummary{}, errors.New("playlist not found or unavailable")
	}

	cover := header.Banner.HeroPlaylistThumbnailRenderer.Thumbnail.url()
	if cover == "" {
		if list, ok := res.firstList(); ok && len(list.Contents) > 0 {
			cover = list.Contents[0].PlaylistVideoRenderer.Thumbnail.url()
		}
	}
	if cover == "" {
		if list, ok := playlistList(data); ok && len(list.Contents) > 0 {
			cover = list.Contents[0].PlaylistVideoRenderer.Thumbnail.url()
		}
	}
	if cover == "" {
		if songs, _ := webPlaylistPage(data); len(songs) > 0 && len(songs[0].Album.Covers) > 0 {
			cover = songs[0].Album.Covers[0]
		}
	}

	return music.PlaylistSummary{
		Name:     header.Title.first(),
		Cover:    cover,
		Subtitle: header.OwnerText.first(),
	}, nil
}

func (s *Service) GetPlaylistSummary(id string) (music.PlaylistSummary, error) {
	playlistID := normalizePlaylistId(id)
	if playlistID == "" {
		return music.PlaylistSummary{}, errors.New("invalid playlist id")
	}
	summary, err := fetchPlaylistSummary(playlistID)
	if err != nil {
		return music.PlaylistSummary{}, err
	}
	summary.Id = music.YouTubePrefix + playlistID
	return summary, nil
}

func (s *Service) GetFullPlaylist(id string) (music.Playlist, error) {
	playlistID := normalizePlaylistId(id)
	if playlistID == "" {
		return music.Playlist{}, errors.New("invalid playlist id")
	}
	return kv.Cached(playlistBucket, "playlist:v3:"+playlistID, func() (music.Playlist, error) {
		return fetchFullPlaylist(playlistID)
	})
}

func (s *Service) RevalidateFullPlaylist(id string) error {
	return playlistBucket.Delete("playlist:v3:" + normalizePlaylistId(id))
}

func (p compactPlaylistRenderer) summary() (music.PlaylistSummary, bool) {
	if p.PlaylistId == "" {
		return music.PlaylistSummary{}, false
	}
	return music.PlaylistSummary{
		Id:       music.YouTubePrefix + p.PlaylistId,
		Name:     p.Title.first(),
		Cover:    p.Thumbnail.url(),
		Subtitle: p.ByLine.first(),
	}, true
}

func extractSongs(list playlistVideoListRenderer, playlist string) []music.Song {
	songs := make([]music.Song, 0, len(list.Contents))
	for _, item := range list.Contents {
		if song, ok := videoToSong(item.PlaylistVideoRenderer, playlist); ok {
			songs = append(songs, song)
		}
	}
	return songs
}

func videoToSong(v playlistVideoRenderer, playlist string) (music.Song, bool) {
	id := strings.TrimSpace(v.VideoId)
	if id == "" {
		return music.Song{}, false
	}

	title := v.Title.first()

	// Skip deleted, private or otherwise unplayable entries instead of listing
	// them. Such entries don't always carry isPlayable, so the placeholder title
	// is checked too.
	if (v.IsPlayable != nil && !*v.IsPlayable) || unavailableTitle(title) {
		return music.Song{}, false
	}

	artist := v.ByLine.first()
	if artist == "" {
		artist = "YouTube"
	}

	album := playlist
	if album == "" {
		album = "YouTube"
	}

	duration, _ := v.LengthSeconds.Int64()

	return music.Song{
		Id:        music.YouTubePrefix + id,
		Url:       "https://www.youtube.com/watch?v=" + id,
		Name:      title,
		Duration:  int(duration),
		Share:     music.Share{Id: music.YouTubePrefix + id, Url: "https://www.youtube.com/watch?v=" + id},
		Album:     music.Album{Title: album, Covers: []string{v.Thumbnail.url()}},
		Artists:   []music.Artist{{Id: v.ByLine.browseID(), Name: artist, Source: "youtube"}},
		YoutubeId: id,
	}, true
}

// unavailableTitle reports whether a playlist entry is a YouTube placeholder
// for a removed video ("[Deleted video]", "[Private video]", ...).
func unavailableTitle(title string) bool {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "[deleted video]", "[private video]", "[unavailable video]", "[video unavailable]":
		return true
	}
	return false
}

func (r playlistBrowseResponse) headerRenderer() (playlistHeaderRenderer, bool) {
	h := r.Header.PlaylistHeaderRenderer
	return h, h.Title.first() != ""
}

func (r playlistBrowseResponse) firstList() (playlistVideoListRenderer, bool) {
	tabs := r.Contents.SingleColumnBrowseResultsRenderer.Tabs
	if len(tabs) == 0 {
		return playlistVideoListRenderer{}, false
	}
	contents := tabs[0].TabRenderer.Content.SectionListRenderer.Contents
	if len(contents) == 0 {
		return playlistVideoListRenderer{}, false
	}
	list := contents[0].PlaylistVideoListRenderer
	return list, len(list.Contents) > 0
}

// playlistPage finds the playlist's videos and their continuation anywhere in a
// browse response. The anonymous clients answer the shape firstList knows; the
// web client, which is the one that opens a private playlist, nests the same
// renderer deeper and moves it between releases, so it is found by walking
// instead of by path.
func playlistPage(data []byte) (list playlistVideoListRenderer, next string, ok bool) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return playlistVideoListRenderer{}, "", false
	}

	walkPlaylists(root, func(raw any) {
		if ok {
			return
		}
		body, err := json.Marshal(raw)
		if err != nil {
			return
		}
		var candidate playlistVideoListRenderer
		if err := json.Unmarshal(body, &candidate); err != nil {
			return
		}
		if len(candidate.Contents) == 0 {
			return
		}
		list, ok = candidate, true
		next = firstContinuationToken(raw)
		if next == "" {
			next = candidate.next()
		}
	})
	if !ok {
		next = firstContinuationToken(root)
	}
	return list, next, ok
}

// playlistList finds the playlist's videos anywhere in a browse response.
func playlistList(data []byte) (playlistVideoListRenderer, bool) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return playlistVideoListRenderer{}, false
	}

	var list playlistVideoListRenderer
	found := false
	walkPlaylists(root, func(raw any) {
		if found {
			return
		}
		body, err := json.Marshal(raw)
		if err != nil {
			return
		}
		var candidate playlistVideoListRenderer
		if err := json.Unmarshal(body, &candidate); err != nil {
			return
		}
		if len(candidate.Contents) > 0 {
			list, found = candidate, true
		}
	})
	return list, found
}

// playlistNext finds the continuation of the list playlistList answered.
func playlistNext(data []byte) string {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return ""
	}

	next := ""
	walkPlaylists(root, func(raw any) {
		if next == "" {
			next = firstContinuationToken(raw)
		}
	})
	if next == "" {
		// No list object: the continuation response carries the token on its own.
		next = firstContinuationToken(root)
	}
	return next
}

func firstContinuationToken(v any) string {
	switch val := v.(type) {
	case map[string]any:
		// The anonymous shape names it "continuation"; the web shape wraps it in
		// a continuationCommand whose field is "token".
		for _, key := range []string{"continuation", "token"} {
			if token, ok := val[key].(string); ok && token != "" {
				return token
			}
		}
		for _, vv := range val {
			if token := firstContinuationToken(vv); token != "" {
				return token
			}
		}
	case []any:
		for _, vv := range val {
			if token := firstContinuationToken(vv); token != "" {
				return token
			}
		}
	}
	return ""
}

// walkPlaylists calls visit with every playlistVideoListRenderer object in the
// tree, wherever it is nested.
func walkPlaylists(v any, visit func(any)) {
	switch val := v.(type) {
	case map[string]any:
		for _, key := range []string{"playlistVideoListRenderer", "playlistVideoListContinuation"} {
			if raw, has := val[key]; has {
				visit(raw)
			}
		}
		for _, vv := range val {
			walkPlaylists(vv, visit)
		}
	case []any:
		for _, vv := range val {
			walkPlaylists(vv, visit)
		}
	}
}
