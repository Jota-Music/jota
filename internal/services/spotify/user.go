package spotify

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"jota/server/internal/music"
)

func (s *SpotifyService) GetUserPlaylists(user string) ([]music.PlaylistSummary, error) {
	url := "https://spclient.wg.spotify.com/user-profile-view/v3/profile/" + user + "/playlists?offset=0&limit=200&market=ES"

	token, err := s.GetToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/x-protobuf")
	req.Header.Set("authorization", "Bearer "+token.AccessToken)
	req.Header.Set("app-platform", "WebPlayer")
	req.Header.Set("spotify-app-version", "1.2.89.334.g2fd3e6cd")
	req.Header.Set("referer", "https://open.spotify.com/")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return parseUserPlaylists(body), nil
}

// ======================================================
// CLEAN STRING (FIX REAL UTF-8 + PROTOBUF GARBAGE)
// ======================================================
func cleanString(s string) string {
	s = strings.Map(func(r rune) rune {
		// elimina control chars + replacement char + invalids
		if r < 32 || r == 127 || r == '\uFFFD' {
			return -1
		}
		return r
	}, s)

	s = strings.ReplaceAll(s, "\u0000", "")

	// normaliza espacios
	s = strings.Join(strings.Fields(s), " ")

	return strings.TrimSpace(s)
}

// ======================================================
// MAIN PARSER
// ======================================================
func parseUserPlaylists(data []byte) []music.PlaylistSummary {
	raw := string(data)

	var out []music.PlaylistSummary

	const playlistPrefix = "spotify:playlist:"
	const mosaicPrefix = "spotify:mosaic:"

	for {
		i := strings.Index(raw, playlistPrefix)
		if i == -1 {
			break
		}

		raw = raw[i+len(playlistPrefix):]

		// -------------------------
		// ID
		// -------------------------
		id := raw
		if j := strings.IndexAny(id, "�'* "); j != -1 {
			id = id[:j]
		}
		if len(id) > 22 {
			id = id[:22]
		}

		rest := raw[len(id):]

		// cortar SOLO esta entrada
		nextEntry := strings.Index(rest, playlistPrefix)
		if nextEntry == -1 {
			nextEntry = len(rest)
		}
		entry := rest[:nextEntry]

		// -------------------------
		// NAME (FIX COMPLETO)
		// -------------------------
		name := extractBetween(entry, []string{
			mosaicPrefix,
			"spotify:user:",
			"https://",
			"�",
			"*",
			"'",
		})
		name = cleanString(name)

		// -------------------------
		// MOSAIC
		// -------------------------
		mosaic := ""
		if i := strings.Index(entry, mosaicPrefix); i != -1 {
			m := entry[i+len(mosaicPrefix):]
			m = cutUntilMarker(m)
			mosaic = buildMosaicURL(mosaicPrefix + m)
		}

		// -------------------------
		// COVER
		// -------------------------
		cover := ""
		if i := strings.Index(entry, "https://"); i != -1 {
			c := entry[i:]
			cover = cleanString(cutUntilMarker(c))
		}

		// avanzar
		raw = rest

		out = append(out, music.PlaylistSummary{
			Id:     id,
			Name:   name,
			Mosaic: mosaic,
			Cover:  cover,
		})
	}

	return out
}

// ======================================================
// HELPERS
// ======================================================
func extractBetween(s string, stop []string) string {
	end := len(s)

	for _, m := range stop {
		if i := strings.Index(s, m); i != -1 && i < end {
			end = i
		}
	}

	return s[:end]
}

func cutUntilMarker(s string) string {
	end := len(s)

	markers := []string{
		"spotify:playlist:",
		"spotify:mosaic:",
		"spotify:user:",
		"�",
		"*",
		"'",
	}

	for _, m := range markers {
		if i := strings.Index(s, m); i != -1 && i < end {
			end = i
		}
	}

	return s[:end]
}

// ======================================================
// MOSAIC BUILDER (CLEAN + SAFE)
// ======================================================
func buildMosaicURL(entry string) string {
	const prefix = "spotify:mosaic:"

	i := strings.Index(entry, prefix)
	if i == -1 {
		return ""
	}

	raw := entry[i+len(prefix):]

	for _, cut := range []string{
		"spotify:",
		"https://",
		"�",
		"*",
		"'",
	} {
		if j := strings.Index(raw, cut); j != -1 {
			raw = raw[:j]
		}
	}

	raw = cleanString(raw)

	if raw == "" {
		return ""
	}

	raw = strings.ReplaceAll(raw, ":", "")

	return "https://mosaic.scdn.co/300/" + raw
}
