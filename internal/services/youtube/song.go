package youtube

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
)

func GetSong(id, search string) (string, error) {
	if youtubeId, err := youtubeSourceBucket.GetString(id); err == nil {
		return youtubeId, nil
	}

	if search == "" {
		return "", errors.New("no search query provided — use ?search= parameter")
	}

	searchQuery := strings.ReplaceAll(search, " ", "+")
	url := fmt.Sprintf("https://music.youtube.com/search?q=%s", searchQuery)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:149.0) Gecko/20100101 Firefox/149.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("search response unreadable: %w", err)
	}

	re := regexp.MustCompile(`videoId\\x22:\\x22(.+?)\\x22`)
	matches := re.FindStringSubmatch(string(body))
	var videoId string
	if len(matches) > 1 {
		videoId = matches[1]
	}

	if videoId == "" {
		bin, err := Ensure()
		if err != nil {
			return "", fmt.Errorf("yt-dlp not available: %w", err)
		}

		ytSearch := fmt.Sprintf("ytsearch1:%s", search)
		args := append(cookiesArgs(), "--skip-download", "--print", "%(id)s", ytSearch)
		cmd := exec.Command(bin, args...)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			searchErr := strings.TrimSpace(stderr.String())
			switch {
			case strings.Contains(searchErr, "Sign in to confirm") || strings.Contains(searchErr, "bot"):
				if HasCookies() {
					return "", fmt.Errorf("YouTube search blocked — cookies may be expired. Upload fresh cookies via POST /api/user/cookies")
				}
				return "", fmt.Errorf("YouTube search blocked — upload cookies via POST /api/user/cookies. See https://github.com/yt-dlp/yt-dlp/wiki/Extractors#exporting-youtube-cookies")
			default:
				return "", fmt.Errorf("search failed: %s", searchErr)
			}
		}

		output := strings.TrimSpace(stdout.String())
		if output != "" && !strings.Contains(stderr.String(), "ERROR") {
			videoId = output
		}
	}

	if videoId == "" {
		return "", fmt.Errorf("no results for: %s", search)
	}

	if err := youtubeSourceBucket.SetString(id, videoId); err != nil {
		return "", fmt.Errorf("cache error: %w", err)
	}

	return videoId, nil
}
