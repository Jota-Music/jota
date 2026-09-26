package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
	"net/url"
	"regexp"
	"strings"
)

var youtubeSourceBucket = kv.UseBucket("youtube-source")

const searchEndpoint = "https://www.youtube.com/youtubei/v1/search"

// search runs an innertube search with an optional filter param and decodes
// the shared response envelope. The music endpoint is used for plain video
// search; the web endpoint for the playlist/channel filtered queries.
func search(url, query, params string) (searchResponse, error) {
	payload := map[string]any{
		"query":          query,
		"context":        map[string]any{"client": clientContext(preferredClient)},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}
	if params != "" {
		payload["params"] = params
	}

	data, err := retryRequest(preferredClient, url, payload, 3)
	if err != nil {
		return searchResponse{}, fmt.Errorf("search request failed: %w", err)
	}

	var sr searchResponse
	if err := json.Unmarshal(data, &sr); err != nil {
		return searchResponse{}, fmt.Errorf("invalid JSON: %w", err)
	}
	return sr, nil
}

func (s *Service) Search(query string) ([]Video, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}

	if id, ok := videoIdFromQuery(query); ok {
		if v, err := fetchVideo(id); err == nil {
			return []Video{v}, nil
		}
	}

	sr, err := search("https://music.youtube.com/youtubei/v1/search", query, "")
	if err != nil {
		return nil, err
	}
	return extractVideos(sr), nil
}

func extractVideos(sr searchResponse) []Video {
	videos := make([]Video, 0)
	for _, section := range sr.Contents.SectionListRenderer.Contents {
		for _, item := range section.ItemSectionRenderer.Contents {
			v := item.CompactVideoRenderer
			if v.VideoID == "" {
				continue
			}
			videos = append(videos, Video{
				ID:        v.VideoID,
				Title:     v.Title.first(),
				Author:    v.ByLine.first(),
				ChannelId: v.ByLine.browseID(),
				Duration:  v.duration(),
			})
		}
	}
	return videos
}

var videoIdPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// videoIdFromQuery extracts a video ID when the query is a YouTube video URL
// (watch?v=, youtu.be, /shorts, /embed, /live) or a bare 11-char video ID.
// A `list=` param does not override the video, so shared "song in playlist"
// links resolve to the song.
func videoIdFromQuery(query string) (string, bool) {
	query = strings.TrimSpace(query)
	u, err := url.Parse(query)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		if videoIdPattern.MatchString(query) {
			return query, true
		}
		return "", false
	}

	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	if host == "youtu.be" {
		id := strings.Trim(u.Path, "/")
		if videoIdPattern.MatchString(id) {
			return id, true
		}
	}
	if id := u.Query().Get("v"); videoIdPattern.MatchString(id) {
		return id, true
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 2 && (parts[0] == "shorts" || parts[0] == "embed" || parts[0] == "live") &&
		videoIdPattern.MatchString(parts[1]) {
		return parts[1], true
	}
	return "", false
}

func fetchVideo(id string) (Video, error) {
	pr, err := fetchPlayer(id)
	if err != nil {
		return Video{}, err
	}
	if pr.VideoDetails.Title == "" {
		return Video{}, errors.New("video unavailable")
	}

	return Video{ID: id, Title: pr.VideoDetails.Title, Author: pr.VideoDetails.Author, ChannelId: pr.VideoDetails.ChannelID}, nil
}

func fetchPlayer(id string) (playerResponse, error) {
	payload := map[string]any{
		"videoId":        id,
		"context":        map[string]any{"client": clientContext(preferredClient)},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	data, err := retryRequest(preferredClient, "https://www.youtube.com/youtubei/v1/player", payload, 3)
	if err != nil {
		return playerResponse{}, fmt.Errorf("player request failed: %w", err)
	}

	var pr playerResponse
	if err := json.Unmarshal(data, &pr); err != nil {
		return playerResponse{}, fmt.Errorf("invalid JSON: %w", err)
	}
	return pr, nil
}

// playlistFilter is the innertube search params that restricts results to playlists.
const playlistFilter = "EgIQAw=="

// channelFilter is the innertube search params that restricts results to channels.
const channelFilter = "EgIQAg=="

// playlistIdPattern matches bare playlist IDs (PL…, LL…, FL…, RD…, UU…, OL…, PU…).
var playlistIdPattern = regexp.MustCompile(`^(?:PL|LL|FL|RD|UU|OL|PU)[A-Za-z0-9_-]{10,}$`)

// playlistIdFromQuery extracts a playlist ID when the query is a YouTube URL with
// a list= param or a bare playlist ID, so pasting a link resolves directly.
func playlistIdFromQuery(query string) string {
	query = strings.TrimSpace(query)
	if u, err := url.Parse(query); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		if id := u.Query().Get("list"); id != "" {
			return normalizePlaylistId(id)
		}
	}
	if playlistIdPattern.MatchString(query) {
		return normalizePlaylistId(query)
	}
	return ""
}

func (s *Service) SearchPlaylists(query string) ([]music.PlaylistSummary, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}

	if id := playlistIdFromQuery(query); id != "" {
		if summary, err := fetchPlaylistSummary(id); err == nil {
			summary.Id = "youtube:" + id
			return []music.PlaylistSummary{summary}, nil
		}
	}

	sr, err := search(searchEndpoint, query, playlistFilter)
	if err != nil {
		return nil, err
	}
	return extractPlaylists(sr), nil
}

func extractChannels(res searchResponse) []music.ChannelInfo {
	out := make([]music.ChannelInfo, 0)
	for _, section := range res.Contents.SectionListRenderer.Contents {
		for _, item := range section.ItemSectionRenderer.Contents {
			if info, ok := item.CompactChannelRenderer.info(); ok {
				out = append(out, info)
			}
		}
	}
	return out
}

func (c compactChannelRenderer) info() (music.ChannelInfo, bool) {
	if c.ChannelId == "" {
		return music.ChannelInfo{}, false
	}
	return music.ChannelInfo{
		Id:     c.ChannelId,
		Name:   c.Title.first(),
		Avatar: c.Thumbnail.url(),
	}, true
}

func (s *Service) SearchChannels(query string) ([]music.ChannelInfo, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}

	sr, err := search(searchEndpoint, query, channelFilter)
	if err != nil {
		return nil, err
	}
	return extractChannels(sr), nil
}

func extractPlaylists(res searchResponse) []music.PlaylistSummary {
	out := make([]music.PlaylistSummary, 0)
	for _, section := range res.Contents.SectionListRenderer.Contents {
		for _, item := range section.ItemSectionRenderer.Contents {
			if summary, ok := item.CompactPlaylistRenderer.summary(); ok {
				out = append(out, summary)
			}
		}
	}
	return out
}

// candidates returns the videos to try for a song, best match first. A cached
// id stays pinned at the front: the user either picked it by hand or it was
// stored by an earlier resolve, and ranking must not demote it behind a fresh
// automatic pick.
func (s *Service) candidates(cacheKey, search string, song music.Song) ([]Video, error) {
	if search == "" {
		if !isYoutubeId(cacheKey) {
			return nil, errors.New("no search query and not a youtube id")
		}
		return []Video{{ID: cacheKey}}, nil
	}

	videos, err := s.Search(search)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var (
		pinned []Video
		found  []Video
		seen   = map[string]bool{}
	)
	if cached, err := youtubeSourceBucket.GetString(cacheKey); err == nil && cached != "" {
		pinned = append(pinned, Video{ID: cached})
		seen[cached] = true
	}
	for _, v := range videos {
		if v.ID != "" && !seen[v.ID] {
			found = append(found, v)
			seen[v.ID] = true
		}
	}

	if len(pinned)+len(found) == 0 {
		return nil, fmt.Errorf("no results for: %s", search)
	}

	return append(pinned, rank(found, song)...), nil
}
