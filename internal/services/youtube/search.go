package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"jota/server/internal/kv"
	"jota/server/internal/music"
	"net/url"
	"regexp"
	"strings"
)

var youtubeSourceBucket = kv.UseBucket("youtube-source")

func (s *Service) Search(query string) ([]Video, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}

	if id, ok := videoIdFromQuery(query); ok {
		if v, err := fetchVideo(id); err == nil {
			return []Video{v}, nil
		}
	}

	payload := map[string]any{
		"query":          query,
		"context":        map[string]any{"client": clientContext(preferredClient)},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	data, err := retryRequest(preferredClient, "https://music.youtube.com/youtubei/v1/search", payload, true, 3)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	var sr searchResponse
	if err := json.Unmarshal(data, &sr); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	var videos []Video
	for _, section := range sr.Contents.SectionListRenderer.Contents {
		for _, item := range section.ItemSectionRenderer.Contents {
			v := item.CompactVideoRenderer
			if v.VideoID == "" {
				continue
			}
			title := ""
			if len(v.Title.Runs) > 0 {
				title = v.Title.Runs[0].Text
			}
			videos = append(videos, Video{ID: v.VideoID, Title: title})
		}
	}

	return videos, nil
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
	payload := map[string]any{
		"videoId":        id,
		"context":        map[string]any{"client": clientContext(preferredClient)},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	data, err := retryRequest(preferredClient, "https://www.youtube.com/youtubei/v1/player", payload, true, 3)
	if err != nil {
		return Video{}, fmt.Errorf("player request failed: %w", err)
	}

	var pr playerResponse
	if err := json.Unmarshal(data, &pr); err != nil {
		return Video{}, fmt.Errorf("invalid JSON: %w", err)
	}
	if pr.VideoDetails.Title == "" {
		return Video{}, errors.New("video unavailable")
	}

	return Video{ID: id, Title: pr.VideoDetails.Title}, nil
}

// playlistFilter is the innertube search params that restricts results to playlists.
const playlistFilter = "EgIQAw=="

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

	payload := map[string]any{
		"query":          query,
		"params":         playlistFilter,
		"context":        map[string]any{"client": clientContext(preferredClient)},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	data, err := retryRequest(preferredClient, "https://www.youtube.com/youtubei/v1/search", payload, true, 3)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	var res playlistSearchResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return extractPlaylists(res), nil
}

func extractPlaylists(res playlistSearchResponse) []music.PlaylistSummary {
	out := make([]music.PlaylistSummary, 0)
	for _, section := range res.Contents.SectionListRenderer.Contents {
		for _, item := range section.ItemSectionRenderer.Contents {
			p := item.CompactPlaylistRenderer
			if p.PlaylistId == "" {
				continue
			}
			out = append(out, music.PlaylistSummary{
				Id:       music.YouTubePrefix + p.PlaylistId,
				Name:     p.Title.first(),
				Cover:    p.Thumbnail.url(),
				Subtitle: p.ByLine.first(),
			})
		}
	}
	return out
}

func (s *Service) candidates(cacheKey, search string) ([]string, error) {
	if search == "" {
		if !isYoutubeId(cacheKey) {
			return nil, errors.New("no search query and not a youtube id")
		}
		return []string{cacheKey}, nil
	}

	var ids []string
	seen := map[string]bool{}
	if cached, err := youtubeSourceBucket.GetString(cacheKey); err == nil && cached != "" {
		ids = append(ids, cached)
		seen[cached] = true
	}

	videos, err := s.Search(search)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	for _, v := range videos {
		if v.ID != "" && !seen[v.ID] {
			ids = append(ids, v.ID)
			seen[v.ID] = true
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no results for: %s", search)
	}

	return ids, nil
}
