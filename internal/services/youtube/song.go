package youtube

import (
	"bytes"
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

	searchQuery := strings.ReplaceAll(search, " ", "+")
	url := fmt.Sprintf("https://music.youtube.com/search?q=%s", searchQuery)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Imitate browser headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:149.0) Gecko/20100101 Firefox/149.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9")
	req.Header.Set("Sec-GPC", "1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch YouTube Music search: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Match videoId\x22:\x22(videoid)\x22 (escaped JSON in HTML)
	re := regexp.MustCompile(`videoId\\x22:\\x22(.+?)\\x22`)
	matches := re.FindStringSubmatch(string(body))
	var videoId string
	if len(matches) > 1 {
		videoId = matches[1]
	}

	if videoId == "" {
		ytSearch := fmt.Sprintf("ytsearch1:%s", search)
		cmd := exec.Command("yt-dlp", "--skip-download", "--print", "%(id)s", ytSearch)

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("yt-dlp failed: %s", strings.TrimSpace(stderr.String()))
		}

		output := strings.TrimSpace(stdout.String())
		if output != "" && !strings.Contains(stderr.String(), "ERROR") {
			videoId = output
		}
	}

	if videoId == "" {
		return "", fmt.Errorf("no video found for query: %s", search)
	}

	if err := youtubeSourceBucket.SetString(id, videoId); err != nil {
		return "", fmt.Errorf("failed to cache song: %w", err)
	}

	return videoId, nil
}
