package spotify

import (
	"encoding/json"
	"fmt"
	"jota/server/internal/music"
	"strings"
)

type TrackVariables struct {
	URI string `json:"uri"`
}

func spotifyTrackURI(idOrURI string) string {
	s := strings.TrimSpace(idOrURI)
	if strings.HasPrefix(s, "spotify:track:") {
		return s
	}
	return fmt.Sprintf("spotify:track:%s", s)
}

func (s *SpotifyService) GetSong(id string) (music.Song, error) {
	body := map[string]any{
		"variables":     TrackVariables{URI: spotifyTrackURI(id)},
		"operationName": "getTrack",
		"extensions": map[string]any{
			"persistedQuery": map[string]any{
				"version":    1,
				"sha256Hash": "612585ae06ba435ad26369870deaae23b5c8800a256cd8a57e08eddc25a37294",
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return music.Song{}, err
	}

	res, err := s.postGraphQL(payload)
	if err != nil {
		return music.Song{}, err
	}

	var parsed struct {
		Data struct {
			TrackUnion json.RawMessage `json:"trackUnion"`
		} `json:"data"`
	}

	if err := json.Unmarshal(res, &parsed); err != nil {
		return music.Song{}, err
	}

	if len(parsed.Data.TrackUnion) == 0 {
		return music.Song{}, fmt.Errorf("spotify getTrack empty trackUnion for '%s'", id)
	}

	var t Track
	if err := json.Unmarshal(parsed.Data.TrackUnion, &t); err != nil {
		return music.Song{}, err
	}

	return ParseTrackToSong(t)
}

func clipString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
