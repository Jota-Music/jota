package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"jota/server/internal/kv"
	"jota/server/internal/music"
)

var youtubeSourceBucket = kv.UseBucket("youtube-source")

func Search(query string) ([]Video, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}

	payload := map[string]any{
		"query":          query,
		"context":        map[string]any{"client": clientContext()},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	data, err := retryRequest("https://music.youtube.com/youtubei/v1/search", payload, true, 3)
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

// playlistFilter is the innertube search params that restricts results to playlists.
const playlistFilter = "EgIQAw=="

func (s *Service) SearchPlaylists(query string) ([]music.PlaylistSummary, error) {
	if query == "" {
		return nil, errors.New("empty query")
	}

	payload := map[string]any{
		"query":          query,
		"params":         playlistFilter,
		"context":        map[string]any{"client": clientContext()},
		"contentCheckOk": true,
		"racyCheckOk":    true,
	}

	data, err := retryRequest("https://www.youtube.com/youtubei/v1/search", payload, true, 3)
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

func candidates(id, search string) ([]string, error) {
	if search == "" {
		if !isYoutubeId(id) {
			return nil, errors.New("no search query and not a youtube id")
		}
		return []string{id}, nil
	}

	var ids []string
	seen := map[string]bool{}
	if cached, err := youtubeSourceBucket.GetString(id); err == nil && cached != "" {
		ids = append(ids, cached)
		seen[cached] = true
	}

	videos, err := Search(search)
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

func SetYoutubeId(id string, youtubeId string) error {
	// Invalidate cached audio so the new video is resolved on the next request.
	if old, err := youtubeSourceBucket.GetString(id); err == nil && old != "" && old != youtubeId {
		_ = audioBucket.Delete(old)
	}
	_ = audioBucket.Delete("spotify:" + id)
	return youtubeSourceBucket.SetString(id, youtubeId)
}
