package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"jota/server/internal/kv"
	"jota/server/internal/music"
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

func browseData(playlistID, continuation string) ([]byte, error) {
	payload := map[string]any{
		"context": map[string]any{"client": clientContext()},
	}
	if continuation != "" {
		payload["continuation"] = continuation
	} else {
		payload["browseId"] = "VL" + playlistID
	}

	data, err := retryRequest(browseEndpoint, payload, true, 3)
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

	songs := extractSongs(list)
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
		songs = append(songs, extractSongs(list)...)
		continuation = list.next()
	}

	name, cover := "", ""
	if header, ok := res.headerRenderer(); ok {
		name = header.Title.first()
		cover = header.Banner.HeroPlaylistThumbnailRenderer.Thumbnail.url()
	}

	return music.Playlist{
		Name:  name,
		Cover: cover,
		Songs: songs,
		Page: music.Page{
			Size:    len(songs),
			Offset:  0,
			Total:   len(songs),
			HasNext: false,
		},
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
	return kv.Cached(playlistBucket, "playlist:"+playlistID, playlistCacheTTL, func() (music.Playlist, error) {
		return fetchFullPlaylist(playlistID)
	})
}

func (s *Service) GetFullPlaylistNoCache(id string) (music.Playlist, error) {
	playlistID := normalizePlaylistId(id)
	if playlistID == "" {
		return music.Playlist{}, errors.New("invalid playlist id")
	}
	return fetchFullPlaylist(playlistID)
}

func (s *Service) RevalidateFullPlaylist(id string) error {
	return playlistBucket.Delete("playlist:" + normalizePlaylistId(id))
}

func (s *Service) AddPlaylist(id string) (music.PlaylistSummary, error) {
	playlistID := normalizePlaylistId(id)
	if playlistID == "" {
		return music.PlaylistSummary{}, errors.New("invalid playlist id")
	}

	summary, err := fetchPlaylistSummary(playlistID)
	if err != nil {
		return music.PlaylistSummary{}, err
	}
	summary.Id = music.YouTubePrefix + playlistID

	var index []music.PlaylistSummary
	if err := playlistBucket.GetObject("index", &index); err != nil &&
		!errors.Is(err, kv.KeyNotFoundError) && !errors.Is(err, kv.KvNotStartedError) {
		return music.PlaylistSummary{}, err
	}

	for i := range index {
		if index[i].Id == summary.Id {
			return summary, nil
		}
	}

	index = append(index, summary)
	if err := playlistBucket.SetObject("index", index); err != nil {
		return music.PlaylistSummary{}, err
	}
	return summary, nil
}

func (s *Service) Playlists() ([]music.PlaylistSummary, error) {
	var index []music.PlaylistSummary
	if err := playlistBucket.GetObject("index", &index); err != nil {
		if errors.Is(err, kv.KeyNotFoundError) || errors.Is(err, kv.KvNotStartedError) {
			return nil, nil
		}
		return nil, err
	}
	return index, nil
}

func (s *Service) RemovePlaylist(id string) error {
	playlistID := normalizePlaylistId(id)
	if playlistID == "" {
		return errors.New("invalid playlist id")
	}

	var index []music.PlaylistSummary
	if err := playlistBucket.GetObject("index", &index); err != nil &&
		!errors.Is(err, kv.KeyNotFoundError) && !errors.Is(err, kv.KvNotStartedError) {
		return err
	}

	out := index[:0]
	for _, p := range index {
		if p.Id != music.YouTubePrefix+playlistID {
			out = append(out, p)
		}
	}
	_ = playlistBucket.Delete("playlist:" + playlistID)
	return playlistBucket.SetObject("index", out)
}

func extractSongs(list playlistVideoListRenderer) []music.Song {
	songs := make([]music.Song, 0, len(list.Contents))
	for _, item := range list.Contents {
		if song, ok := videoToSong(item.PlaylistVideoRenderer); ok {
			songs = append(songs, song)
		}
	}
	return songs
}

func videoToSong(v playlistVideoRenderer) (music.Song, bool) {
	id := strings.TrimSpace(v.VideoId)
	if id == "" || (v.IsPlayable != nil && !*v.IsPlayable) {
		return music.Song{}, false
	}

	artist := v.ByLine.first()
	if artist == "" {
		artist = "YouTube"
	}

	duration, _ := v.LengthSeconds.Int64()

	return music.Song{
		Id:        music.YouTubePrefix + id,
		Url:       "https://www.youtube.com/watch?v=" + id,
		Name:      v.Title.first(),
		Duration:  int(duration),
		Share:     music.Share{Id: music.YouTubePrefix + id, Url: "https://www.youtube.com/watch?v=" + id},
		Album:     music.Album{Title: "YouTube", Covers: []string{v.Thumbnail.url()}},
		Artists:   []music.Artist{{Name: artist}},
		YoutubeId: id,
	}, true
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
