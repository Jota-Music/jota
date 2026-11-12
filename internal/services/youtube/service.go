package youtube

import (
	"errors"
	"strings"

	"github.com/Jota-Music/jota/internal/music"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// GetAudio returns the audio stream for a known YouTube video ID.
func (s *Service) GetAudio(id string) (music.Audio, error) {
	id = strings.TrimPrefix(id, music.YouTubePrefix)
	if !isYoutubeId(id) {
		return music.Audio{}, errors.New("invalid youtube id")
	}
	audio, err := fetchAudio(id)
	if err != nil {
		return music.Audio{}, err
	}
	return *audio, nil
}

// ResolveAudio returns a stream for any song. When the song already carries a
// linked YouTube ID it is used directly; if that video is unavailable it falls
// back to searching by the song's own metadata.
func (s *Service) ResolveAudio(song music.Song) (music.Audio, error) {
	if song.YoutubeId != "" {
		audio, err := s.GetAudio(song.YoutubeId)
		if err == nil {
			return audio, nil
		}
	}
	return s.searchAudio(song)
}

// Find returns the YouTube video linked to a song without resolving a stream.
// It prefers the id the song carries, then the cached one, and only searches
// when both are missing, recording whatever it settles on for later resolves.
func (s *Service) Find(song music.Song) (string, error) {
	if song.YoutubeId != "" {
		return song.YoutubeId, nil
	}
	if id := pinned(song.Id); id != "" {
		return id, nil
	}
	if id := remembered(song.Id); id != "" {
		return id, nil
	}

	videos, err := s.candidates(song.Id, song)
	if err != nil {
		return "", err
	}
	if len(videos) == 0 {
		return "", errors.New("no youtube video found")
	}
	id := videos[0].ID
	remember(song.Id, id)
	return id, nil
}

func (s *Service) searchAudio(song music.Song) (music.Audio, error) {
	cacheKey := song.Id

	// The linked id is the video an earlier resolve settled on. The pin the
	// user fixed by hand wins over the pick a search made; only the pin is an
	// order, so a resolve never overwrites it. Its stream is cached under the
	// video, so playing it first skips the search, the slowest step of a
	// resolve; searching for alternatives is only worth it once it fails.
	linked := pinned(cacheKey)
	wasPin := linked != ""
	if linked == "" {
		linked = remembered(cacheKey)
	}
	if linked != "" {
		if cached := cachedAudio(linked); cached != nil {
			return *cached, nil
		}
	}

	var lastErr error
	tried := ""
	if linked != "" {
		tried = linked
		audio, err := s.play(cacheKey, linked)
		if err == nil {
			return *audio, nil
		}
		lastErr = err
	}

	videos, err := s.candidates(cacheKey, song)
	if err != nil {
		return music.Audio{}, err
	}

	for _, v := range videos {
		if v.ID == tried {
			continue
		}
		audio, err := s.play(cacheKey, v.ID)
		if err != nil {
			lastErr = err
			continue
		}
		// A pinned video that died is useless: a fallback just played, so the
		// network is fine and the pin, not the connection, was the problem.
		if wasPin {
			_ = forgetPin(cacheKey)
		}
		return *audio, nil
	}

	return music.Audio{}, lastErr
}

// play resolves one candidate and records it as the song's source, so the next
// resolve skips the search. fetchAudio already serves the cached stream when it
// is still valid, so a separate cache read here would only probe googlevideo
// twice for the same URL.
func (s *Service) play(cacheKey, videoID string) (*music.Audio, error) {
	audio, err := fetchAudio(videoID)
	if err != nil {
		return nil, err
	}
	if cacheKey != "" {
		remember(cacheKey, videoID)
	}
	return audio, nil
}

// YoutubeId returns the effective video for a song: the pin the user fixed, or
// the pick an earlier search settled on.
func (s *Service) YoutubeId(cacheKey string) string {
	if v := pinned(cacheKey); v != "" {
		return v
	}
	return remembered(cacheKey)
}

func (s *Service) SetYoutubeId(cacheKey, youtubeId string) error {
	youtubeId = strings.TrimPrefix(strings.TrimSpace(youtubeId), music.YouTubePrefix)
	if youtubeId != "" {
		id, ok := videoIdFromQuery(youtubeId)
		if !ok {
			return errors.New("invalid youtube id")
		}
		youtubeId = id
	}
	return pin(cacheKey, youtubeId)
}

// audioQuery asks YouTube for the song by title, artist and album. The album
// disambiguates a title YouTube spells differently — "Lo que quieras" against
// Spotify's "Lo que quieres" — because the distributor keeps the release name in
// the upload's metadata. audioQueryArtistFirst drops the album: an album that
// shares its name with another track on it buries the song under that track, so
// both queries run together and ranking picks between their merged results.
func audioQuery(song music.Song) string {
	query := song.Name + " " + artistNames(song)
	if album := strings.TrimSpace(song.Album.Title); album != "" {
		query += " " + album
	}
	return strings.TrimSpace(query)
}

func audioQueryArtistFirst(song music.Song) string {
	names := artistNames(song)
	if names == "" {
		return strings.TrimSpace(song.Name)
	}
	return strings.TrimSpace(names + " " + song.Name)
}

// queries returns the distinct searches to run for a song, album-first then
// artist-first. They collapse to one when the song carries no album.
func queries(song music.Song) []string {
	album, artist := audioQuery(song), audioQueryArtistFirst(song)
	if album == "" {
		return nil
	}
	if artist == "" || artist == album {
		return []string{album}
	}
	return []string{album, artist}
}

func artistNames(song music.Song) string {
	names := make([]string, 0, len(song.Artists))
	for _, artist := range song.Artists {
		if artist.Name != "" {
			names = append(names, artist.Name)
		}
	}
	return strings.Join(names, ", ")
}

func isYoutubeId(id string) bool {
	return videoIdPattern.MatchString(id)
}
