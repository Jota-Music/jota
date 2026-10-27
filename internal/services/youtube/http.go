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

// post sends one innertube request signed with raw, or anonymously when raw is
// empty. A jar that is not the stored session goes through here, so the app can
// ask about a browser before adopting it.
func post(c clientConfig, url string, payload map[string]any, raw string) (*http.Response, error) {
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
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-YouTube-Client-Name", fmt.Sprintf("%d", c.ClientName))
	req.Header.Set("X-YouTube-Client-Version", c.Version)
	req.Header.Set("Origin", origin)
	req.Header.Set("Referer", origin+"/")
	if raw != "" {
		if session := sessionVisitor(); session != "" {
			visitor = session
		}
	}
	if visitor != "" {
		req.Header.Set("X-Goog-Visitor-Id", visitor)
	}
	// Signed-in requests carry SAPISIDHASH, which is what unlocks age-restricted
	// videos and avoids the bot check for the account's session.
	if raw != "" {
		for k, v := range authHeadersFor(raw, origin) {
			req.Header.Set(k, v)
		}
	}

	return innertubeClient.Do(req)
}

func retryRequest(c clientConfig, url string, payload map[string]any, retries int) ([]byte, error) {
	return retry(c, url, payload, retries, cookies())
}

// retryAnonymous is retryRequest without the account session.
func retryAnonymous(c clientConfig, url string, payload map[string]any, retries int) ([]byte, error) {
	return retry(c, url, payload, retries, "")
}

// retryAs is retryRequest signed with a jar that is not the stored session.
func retryAs(raw string, c clientConfig, url string, payload map[string]any, retries int) ([]byte, error) {
	return retry(c, url, payload, retries, raw)
}

func retry(c clientConfig, url string, payload map[string]any, retries int, raw string) ([]byte, error) {
	var lastErr error
	for i := range retries {
		resp, err := post(c, url, payload, raw)
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
