package youtube

import (
	"errors"
	"strings"

	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/music"
)

// GetSong resolves a single video by ID to its full metadata.
func (s *Service) GetSong(id string) (music.Song, error) {
	id = strings.TrimPrefix(strings.TrimSpace(id), music.YouTubePrefix)
	if !isYoutubeId(id) {
		return music.Song{}, errors.New("invalid youtube id")
	}

	return kv.Cached(youtubeSourceBucket, "song:v2:"+id, func() (music.Song, error) {
		return fetchSong(id)
	})
}

func fetchSong(id string) (music.Song, error) {
	pr, err := fetchPlayer(id)
	if err != nil {
		return music.Song{}, err
	}
	if pr.VideoDetails.Title == "" {
		return music.Song{}, errors.New("video unavailable")
	}

	author := pr.VideoDetails.Author
	if author == "" {
		author = "YouTube"
	}

	duration, _ := pr.VideoDetails.LengthSeconds.Int64()
	watchURL := "https://www.youtube.com/watch?v=" + id

	return music.Song{
		Id:        music.YouTubePrefix + id,
		Url:       watchURL,
		Name:      pr.VideoDetails.Title,
		Duration:  int(duration),
		Share:     music.Share{Id: music.YouTubePrefix + id, Url: watchURL},
		Album:     music.Album{Title: "YouTube", Covers: []string{pr.VideoDetails.Thumbnail.url()}},
		Artists:   []music.Artist{{Id: pr.VideoDetails.ChannelID, Name: author, Source: "youtube"}},
		YoutubeId: id,
	}, nil
}
