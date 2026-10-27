package youtube

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Jota-Music/jota/internal/music"
)

// webPlaylistPage reads the playlist page the web client answers with, the one
// that comes back for the playlists a session owns. Its videos are
// lockupViewModel items instead of the legacy playlist renderer, and its title
// lives in pageHeaderRenderer.
//
// Only the first page is read: the token continuing it is bound to the page
// that asked for it and cannot be replayed from here.
func webPlaylistPage(data []byte) (songs []music.Song, title string) {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, ""
	}

	// Two passes: the title tags each song as its album, and the walk hits
	// maps in an unpredictable order.
	walkJSON(root, func(node map[string]any) {
		if t, ok := node["pageTitle"].(string); ok && title == "" {
			title = t
		}
	})
	walkJSON(root, func(node map[string]any) {
		if node["contentType"] != "LOCKUP_CONTENT_TYPE_VIDEO" {
			return
		}
		if song, ok := lockupSong(node, title); ok {
			songs = append(songs, song)
		}
	})
	return songs, title
}

// pageTitle returns the title a modern page header carries, if any.
func pageTitle(data []byte) string {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return ""
	}
	title := ""
	walkJSON(root, func(node map[string]any) {
		if title != "" {
			return
		}
		if t, ok := node["pageTitle"].(string); ok {
			title = t
		}
	})
	return title
}

// walkJSON visits every map of a decoded JSON tree.
func walkJSON(v any, visit func(map[string]any)) {
	switch val := v.(type) {
	case map[string]any:
		visit(val)
		for _, vv := range val {
			walkJSON(vv, visit)
		}
	case []any:
		for _, vv := range val {
			walkJSON(vv, visit)
		}
	}
}

// lockupSong turns a video lockup into the song the playlist holds.
func lockupSong(m map[string]any, playlist string) (music.Song, bool) {
	id, _ := m["contentId"].(string)
	// A lockup's contentId is a video id here; a playlist id is longer, so the
	// shared pattern keeps the two apart.
	if !videoIdPattern.MatchString(id) {
		return music.Song{}, false
	}

	meta, _ := m["metadata"].(map[string]any)["lockupMetadataViewModel"].(map[string]any)
	title, _ := meta["title"].(map[string]any)["content"].(string)
	if title == "" || unavailableTitle(title) {
		return music.Song{}, false
	}

	artist, artistID := lockupArtist(meta)
	album := playlist
	if album == "" {
		album = "YouTube"
	}

	return music.Song{
		Id:        music.YouTubePrefix + id,
		Url:       "https://www.youtube.com/watch?v=" + id,
		Name:      title,
		Duration:  lockupDuration(m),
		Share:     music.Share{Id: music.YouTubePrefix + id, Url: "https://www.youtube.com/watch?v=" + id},
		Album:     music.Album{Title: album, Covers: []string{lockupCover(m)}},
		Artists:   []music.Artist{{Id: artistID, Name: artist, Source: "youtube"}},
		YoutubeId: id,
	}, true
}

// lockupArtist reads the byline of a lockup: the first part of its first
// metadata row, plus the channel it links to.
func lockupArtist(meta map[string]any) (name, channelID string) {
	rows, _ := meta["metadata"].(map[string]any)["contentMetadataViewModel"].(map[string]any)["metadataRows"].([]any)
	if len(rows) == 0 {
		return "YouTube", ""
	}
	parts, _ := rows[0].(map[string]any)["metadataParts"].([]any)
	if len(parts) == 0 {
		return "YouTube", ""
	}
	name, _ = parts[0].(map[string]any)["text"].(map[string]any)["content"].(string)
	if name == "" {
		name = "YouTube"
	}
	walkJSON(parts[0], func(node map[string]any) {
		if channelID != "" {
			return
		}
		if id, ok := node["browseId"].(string); ok && channelIDPattern.MatchString(id) {
			channelID = id
		}
	})
	return name, channelID
}

// lockupDuration reads the length badge the thumbnail carries ("7:52",
// "1:02:03").
func lockupDuration(m map[string]any) int {
	seconds := 0
	walkJSON(m["contentImage"], func(node map[string]any) {
		if seconds != 0 {
			return
		}
		badge, ok := node["thumbnailBadgeViewModel"].(map[string]any)
		if !ok {
			return
		}
		text, _ := badge["text"].(string)
		seconds = clockSeconds(text)
	})
	return seconds
}

// clockSeconds parses mm:ss and h:mm:ss into seconds, 0 when the text is not a
// running time.
func clockSeconds(text string) int {
	fields := strings.Split(text, ":")
	if len(fields) < 2 || len(fields) > 3 {
		return 0
	}
	total := 0
	for _, field := range fields {
		n, err := strconv.Atoi(field)
		if err != nil || n < 0 {
			return 0
		}
		total = total*60 + n
	}
	return total
}

// lockupCover returns the largest thumbnail of a lockup.
func lockupCover(m map[string]any) string {
	cover := ""
	walkJSON(m["contentImage"], func(node map[string]any) {
		sources, ok := node["sources"].([]any)
		if !ok || len(sources) == 0 {
			return
		}
		last, ok := sources[len(sources)-1].(map[string]any)
		if !ok {
			return
		}
		if url, ok := last["url"].(string); ok && url != "" {
			cover = absolute(url)
		}
	})
	if cover == "" {
		return ""
	}
	return capThumbnail(cover)
}
