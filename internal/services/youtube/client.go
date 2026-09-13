package youtube

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"
)

const defaultAPIKey = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"

const visitorTTL = 5 * time.Minute

var (
	innertubeApiKeyRe = regexp.MustCompile(`"INNERTUBE_API_KEY":"([^"]+)"`)
	visitorDataRe     = regexp.MustCompile(`"VISITOR_DATA":"([^"]+)"`)
)

var (
	client = struct {
		Name        string
		Version     string
		ClientName  int
		UserAgent   string
		DeviceMake  string
		DeviceModel string
		OsName      string
		OsVersion   string
	}{
		Name:        "VISIONOS",
		Version:     "1.02",
		ClientName:  101,
		UserAgent:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_7_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15",
		DeviceMake:  "Apple",
		DeviceModel: "RealityDevice17,1",
		OsName:      "visionOS",
		OsVersion:   "26.5.23O471",
	}

	apiKeyMu      sync.RWMutex
	apiKey        = defaultAPIKey
	visitorDataMu sync.RWMutex
	visitorData   string
	lastFetch     time.Time
)

func currentAPIKey() string {
	apiKeyMu.RLock()
	defer apiKeyMu.RUnlock()
	return apiKey
}

func clientContext() map[string]any {
	return map[string]any{
		"clientName":    client.Name,
		"clientVersion": client.Version,
		"deviceMake":    client.DeviceMake,
		"deviceModel":   client.DeviceModel,
		"osName":        client.OsName,
		"osVersion":     client.OsVersion,
		"hl":            "en",
	}
}

// getVisitorData returns the cached visitor data and API key, refreshing them
// from YouTube's homepage at most once per visitorTTL. Freshness is tracked by
// lastFetch regardless of whether the homepage carried VISITOR_DATA, so a
// homepage variant without it does not trigger a full fetch on every request.
func getVisitorData() (string, string, error) {
	visitorDataMu.RLock()
	if !lastFetch.IsZero() && time.Since(lastFetch) < visitorTTL {
		vd := visitorData
		visitorDataMu.RUnlock()
		return vd, currentAPIKey(), nil
	}
	visitorDataMu.RUnlock()

	visitorDataMu.Lock()
	defer visitorDataMu.Unlock()
	if !lastFetch.IsZero() && time.Since(lastFetch) < visitorTTL {
		return visitorData, currentAPIKey(), nil
	}

	req, err := http.NewRequest("GET", "https://www.youtube.com/", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", client.UserAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if m := innertubeApiKeyRe.FindSubmatch(body); len(m) > 1 {
		apiKeyMu.Lock()
		apiKey = string(m[1])
		apiKeyMu.Unlock()
	}

	if m := visitorDataRe.FindSubmatch(body); len(m) > 1 {
		visitorData = string(m[1])
	}
	lastFetch = time.Now()

	return visitorData, currentAPIKey(), nil
}
