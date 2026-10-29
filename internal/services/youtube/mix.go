package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Jota-Music/jota/internal/music"
)

const nextEndpoint = "https://www.youtube.com/youtubei/v1/next"

// mixSeed returns the video a radio playlist is built around. A watch link
// carries its mix as `list=RD<video id>`, and the browse endpoint answers that
// id with 400: YouTube serves the mix in the watch's next response instead.
//
// shortcut: only RD<video id> mixes are read this way, a radio whose id is not
// seeded by a video (RDCLAK…, RDAMVM…) still goes through browse. Upgrade when
// one of those is reported broken.
func mixSeed(playlistID string) string {
	seed := strings.TrimPrefix(playlistID, "RD")
	if seed == playlistID || !videoIdPattern.MatchString(seed) {
		return ""
	}
	return seed
}

// fetchMix reads a radio playlist. The web client is the one that answers with
// the playlist panel; the anonymous clients return a watch page without it.
func fetchMix(playlistID, seed string) (music.Playlist, error) {
	data, err := postMix(playlistID, seed)
	if err != nil {
		return music.Playlist{}, err
	}
	return parseMix(data)
}

func parseMix(data []byte) (music.Playlist, error) {
	var res mixResponse
	if err := json.Unmarshal(data, &res); err != nil {
		return music.Playlist{}, fmt.Errorf("invalid mix JSON: %w", err)
	}

	panel := res.Contents.TwoColumnWatchNextResults.Playlist.Playlist
	name := panel.Title
	if name == "" {
		name = "Mix"
	}

	songs := make([]music.Song, 0, len(panel.Contents))
	for _, item := range panel.Contents {
		if song, ok := mixToSong(item.PlaylistPanelVideoRenderer, name); ok {
			songs = append(songs, song)
		}
	}
	if len(songs) == 0 {
		return music.Playlist{}, errors.New("mix has no videos")
	}

	return music.Playlist{
		Name:  name,
		Cover: songs[0].Album.Covers[0],
		Owner: panel.LongByLine.first(),
		Songs: songs,
	}, nil
}

// mixSummary is the metadata a saved mix needs, read from the same response.
func mixSummary(playlistID, seed string) (music.PlaylistSummary, error) {
	playlist, err := fetchMix(playlistID, seed)
	if err != nil {
		return music.PlaylistSummary{}, err
	}
	return music.PlaylistSummary{
		Name:     playlist.Name,
		Cover:    playlist.Cover,
		Subtitle: playlist.Owner,
	}, nil
}

func postMix(playlistID, seed string) ([]byte, error) {
	c := webClient()
	payload := map[string]any{
		"context":    map[string]any{"client": clientContext(c)},
		"videoId":    seed,
		"playlistId": playlistID,
	}
	return browseResult(retryRequest(c, nextEndpoint, payload, 3))
}

func mixToSong(v mixVideo, mix string) (music.Song, bool) {
	id := strings.TrimSpace(v.VideoID)
	if id == "" {
		return music.Song{}, false
	}

	title := v.Title.first()
	if unavailableTitle(title) {
		return music.Song{}, false
	}

	artist := v.ByLine.first()
	if artist == "" {
		artist = "YouTube"
	}

	return music.Song{
		Id:        music.YouTubePrefix + id,
		Url:       "https://www.youtube.com/watch?v=" + id,
		Name:      title,
		Duration:  clockSeconds(v.LengthText.first()),
		Share:     music.Share{Id: music.YouTubePrefix + id, Url: "https://www.youtube.com/watch?v=" + id},
		Album:     music.Album{Title: mix, Covers: []string{v.Thumbnail.url()}},
		Artists:   []music.Artist{{Id: v.ByLine.browseID(), Name: artist, Source: "youtube"}},
		YoutubeId: id,
	}, true
}

type mixResponse struct {
	Contents struct {
		TwoColumnWatchNextResults struct {
			Playlist struct {
				Playlist struct {
					Title      string    `json:"title"`
					LongByLine text      `json:"longBylineText"`
					Contents   []mixItem `json:"contents"`
				} `json:"playlist"`
			} `json:"playlist"`
		} `json:"twoColumnWatchNextResults"`
	} `json:"contents"`
}

type mixItem struct {
	PlaylistPanelVideoRenderer mixVideo `json:"playlistPanelVideoRenderer"`
}

type mixVideo struct {
	VideoID    string    `json:"videoId"`
	Title      text      `json:"title"`
	ByLine     text      `json:"longBylineText"`
	LengthText text      `json:"lengthText"`
	Thumbnail  thumbnail `json:"thumbnail"`
}
