package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func doRequest(url string, payload map[string]any, useVisitor bool) (*http.Response, error) {
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", client.UserAgent)
	req.Header.Set("X-YouTube-Client-Name", fmt.Sprintf("%d", client.ClientName))
	req.Header.Set("X-YouTube-Client-Version", client.Version)
	req.Header.Set("Origin", "https://www.youtube.com")
	req.Header.Set("Referer", "https://www.youtube.com/")
	if useVisitor && visitor != "" {
		req.Header.Set("X-Goog-Visitor-Id", visitor)
	}

	return http.DefaultClient.Do(req)
}

func retryRequest(url string, payload map[string]any, useVisitor bool, retries int) ([]byte, error) {
	var lastErr error
	for i := range retries {
		resp, err := doRequest(url, payload, useVisitor)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(100+(i*200)) * time.Millisecond)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
			time.Sleep(time.Duration(500+(i*500)) * time.Millisecond)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(100+(i*100)) * time.Millisecond)
			continue
		}

		return data, nil
	}
	return nil, lastErr
}
