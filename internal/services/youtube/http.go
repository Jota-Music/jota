package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// innertubeClient bounds every API call. http.DefaultClient has no timeout, so
// a hung connection would stall a resolve (and playback) forever.
var innertubeClient = &http.Client{Timeout: 15 * time.Second}

func doRequest(c clientConfig, url string, payload map[string]any, useVisitor bool) (*http.Response, error) {
	visitor, key, err := getVisitorData()
	if err != nil {
		return nil, fmt.Errorf("getting visitor data: %w", err)
	}

	if key == "" {
		key = defaultAPIKey
	}

	fullURL := url + "?key=" + key
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	origin := originOf(url)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-YouTube-Client-Name", fmt.Sprintf("%d", c.ClientName))
	req.Header.Set("X-YouTube-Client-Version", c.Version)
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", origin+"/")
	if useVisitor && visitor != "" {
		req.Header.Set("X-Goog-Visitor-Id", visitor)
	}
	// Signed-in requests carry SAPISIDHASH, which is what unlocks age-restricted
	// videos and avoids the bot check for the account's session.
	for k, v := range authHeaders(origin) {
		req.Header.Set(k, v)
	}

	return innertubeClient.Do(req)
}

func retryRequest(c clientConfig, url string, payload map[string]any, useVisitor bool, retries int) ([]byte, error) {
	var lastErr error
	for i := range retries {
		resp, err := doRequest(c, url, payload, useVisitor)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(100+(i*200)) * time.Millisecond)
			continue
		}

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			time.Sleep(time.Duration(500+(i*500)) * time.Millisecond)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(100+(i*100)) * time.Millisecond)
			continue
		}

		adoptVisitor(data)
		return data, nil
	}
	return nil, lastErr
}
