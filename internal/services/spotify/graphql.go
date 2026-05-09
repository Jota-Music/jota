package spotify

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// postGraphQL sends a persisted-query body to the partner API. On 401 it clears
// cached and in-memory tokens and retries once with a freshly obtained bearer.
func (s *SpotifyService) postGraphQL(payload []byte) ([]byte, error) {
	for attempt := 0; attempt < 2; attempt++ {
		token, err := s.GetToken()
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest("POST", SPOTIFY_API_ENDPOINT, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("authorization", "Bearer "+token.AccessToken)
		req.Header.Set("User-Agent", "Mozilla/5.0")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}

		if resp.StatusCode == http.StatusUnauthorized {
			if attempt == 0 {
				s.invalidateToken()
				continue
			}
			return nil, fmt.Errorf("spotify HTTP status '%s' body '%s'", resp.Status, clipString(string(data), 800))
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("spotify HTTP status '%s' body '%s'", resp.Status, clipString(string(data), 800))
		}
		return data, nil
	}
	return nil, fmt.Errorf("spotify: postGraphQL retry exhausted")
}
