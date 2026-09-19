package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
	"regexp"
	"strings"
	"time"
)

var playlistBucket = kv.UseBucket("youtube-playlists")

const playlistCacheTTL = 12 * time.Hour

const browseEndpoint = "https://www.youtube.com/youtubei/v1/browse"

func normalizePlaylistId(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, music.YouTubePrefix)
	id = strings.TrimPrefix(id, "VL")
	return id
}

func postBrowse(payload map[string]any) ([]byte, error) {
	data, err := retryRequest(preferredClient, browseEndpoint, payload, 3)
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
		return nil, fmt.Errorf("browse error: %s", res.Error.Message)
	}

	return data, nil
}

func browseData(playlistID, continuation string) ([]byte, error) {
	payload := map[string]any{
		"context": map[string]any{"client": clientContext(preferredClient)},
	}
	if continuation != "" {
		payload["continuation"] = continuation
	} else {
		payload["browseId"] = "VL" + playlistID
	}

	return postBrowse(payload)
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

	return kv.Cached(playlistBucket, "channel-id:"+channel, playlistCacheTTL, func() (string, error) {
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
	return kv.Cached(playlistBucket, "channel:playlists:"+channelID, playlistCacheTTL, func() ([]music.PlaylistSummary, error) {
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
	return kv.Cached(playlistBucket, "channel:videos:"+channelID, playlistCacheTTL, func() ([]music.Song, error) {
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
	return kv.Cached(playlistBucket, "channel:info:"+channelID, playlistCacheTTL, func() (music.ChannelInfo, error) {
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
	})
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

	list, ok := res.firstList()
	if !ok {
		return music.Playlist{}, errors.New("playlist not found or has no videos")
	}

	name, cover, owner, ownerID := "", "", "", ""
	if header, ok := res.headerRenderer(); ok {
		name = header.Title.first()
		cover = header.Banner.HeroPlaylistThumbnailRenderer.Thumbnail.url()
		owner = header.OwnerText.first()
		ownerID = header.OwnerText.browseID()
	}

	songs := extractSongs(list, name)
	continuation := list.next()

	for pages := 1; continuation != ""; pages++ {
		// ponytail: cap the loop so a repeated continuation token can't spin forever
		if pages > 500 {
			return music.Playlist{}, errors.New("playlist too large or malformed continuation")
		}

		data, err := browseData(playlistID, continuation)
		if err != nil {
			return music.Playlist{}, err
		}

		var cont playlistContinuationResponse
		if err := json.Unmarshal(data, &cont); err != nil {
			return music.Playlist{}, fmt.Errorf("invalid continuation JSON: %w", err)
		}

		list := cont.ContinuationContents.PlaylistVideoListContinuation
		songs = append(songs, extractSongs(list, name)...)
		continuation = list.next()
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
	if !ok {
		return music.PlaylistSummary{}, errors.New("playlist not found or unavailable")
	}

	cover := header.Banner.HeroPlaylistThumbnailRenderer.Thumbnail.url()
	if cover == "" {
		if list, ok := res.firstList(); ok && len(list.Contents) > 0 {
			cover = list.Contents[0].PlaylistVideoRenderer.Thumbnail.url()
		}
	}

	return music.PlaylistSummary{
		Name:     header.Title.first(),
		Cover:    cover,
		Subtitle: header.OwnerText.first(),
	}, nil
}

func (s *Service) GetFullPlaylist(id string) (music.Playlist, error) {
	playlistID := normalizePlaylistId(id)
	if playlistID == "" {
		return music.Playlist{}, errors.New("invalid playlist id")
	}
	return kv.Cached(playlistBucket, "playlist:v3:"+playlistID, playlistCacheTTL, func() (music.Playlist, error) {
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
