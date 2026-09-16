package youtube

import (
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

// defaultAPIKey is Google's public InnerTube key, the same one shipped in the
// YouTube web client. Set YOUTUBE_API_KEY to override it (both here and the
// value scraped from YouTube) with your own key.
const defaultAPIKey = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"

var apiKeyEnv = os.Getenv("YOUTUBE_API_KEY")

const (
	visitorTTL       = 5 * time.Minute
	visitorFetchWait = 10 * time.Second
	visitorRetries   = 3
)

var (
	innertubeApiKeyRe = regexp.MustCompile(`"INNERTUBE_API_KEY":"([^"]+)"`)
	visitorDataRe     = regexp.MustCompile(`"VISITOR_DATA":"([^"]+)"`)
)

var visitorClient = &http.Client{Timeout: visitorFetchWait}

type clientConfig struct {
	Name        string
	Version     string
	ClientName  int
	UserAgent   string
	DeviceMake  string
	DeviceModel string
	OsName      string
	OsVersion   string
}

// preferredClient is tried first. Some clients hand out googlevideo URLs that
// 403 on a plain/HEAD request and only answer bounded Range requests, which
// GStreamer/WebKitGTK does not send first; fallbackClients cover that case.
var preferredClient = clientConfig{
	Name:        "VISIONOS",
	Version:     "1.02",
	ClientName:  101,
	UserAgent:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_7_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15",
	DeviceMake:  "Apple",
	DeviceModel: "RealityDevice17,1",
	OsName:      "visionOS",
	OsVersion:   "26.5.23O471",
}

var fallbackClients = []clientConfig{
	{
		Name:        "ANDROID_VR",
		Version:     "1.62.27",
		ClientName:  28,
		UserAgent:   "com.google.android.apps.youtube.vr.oculus/1.62.27 (Linux; U; Android 12; GB) gzip",
		DeviceMake:  "Oculus",
		DeviceModel: "Quest 3",
		OsName:      "Android",
		OsVersion:   "12",
	},
	{
		Name:        "IOS",
		Version:     "20.10.4",
		ClientName:  5,
		UserAgent:   "com.google.ios.youtube/20.10.4 (iPhone16,2; U; CPU iOS 18_3_2 like Mac OS X)",
		DeviceMake:  "Apple",
		DeviceModel: "iPhone16,2",
		OsName:      "iOS",
		OsVersion:   "18.3.2.22D82",
	},
}

func allClients() []clientConfig {
	return append([]clientConfig{preferredClient}, fallbackClients...)
}

var (
	apiKeyMu      sync.RWMutex
	apiKey        = initialAPIKey()
	visitorDataMu sync.RWMutex
	visitorData   string
	lastFetch     time.Time
)

func initialAPIKey() string {
	if apiKeyEnv != "" {
		return apiKeyEnv
	}
	return defaultAPIKey
}

func currentAPIKey() string {
	apiKeyMu.RLock()
	defer apiKeyMu.RUnlock()
	return apiKey
}

func clientContext(c clientConfig) map[string]any {
	return map[string]any{
		"clientName":    c.Name,
		"clientVersion": c.Version,
		"deviceMake":    c.DeviceMake,
		"deviceModel":   c.DeviceModel,
		"osName":        c.OsName,
		"osVersion":     c.OsVersion,
		"hl":            "en",
	}
}

// invalidateVisitor drops the cached visitor token so the next request
// re-fetches it from the homepage. YouTube's bot check serves LOGIN_REQUIRED
// (no stream URLs) once a token is stale or used up.
func invalidateVisitor() {
	visitorDataMu.Lock()
	visitorData = ""
	lastFetch = time.Time{}
	visitorDataMu.Unlock()
}

// getVisitorData returns the cached visitor data and API key, refreshing them
// from YouTube's homepage at most once per visitorTTL. Freshness is tracked by
// lastFetch regardless of whether the homepage carried VISITOR_DATA, so a
// homepage variant without it does not trigger a full fetch on every request.
// A failed refresh degrades to the cached values instead of erroring, so a
// flaky homepage never takes playback down.
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

	var body []byte
	fetched := false
	for i := range visitorRetries {
		req, err := http.NewRequest("GET", "https://www.youtube.com/", nil)
		if err != nil {
			log.Printf("youtube: visitor fetch request build failed: %v", err)
			break
		}
		req.Header.Set("User-Agent", preferredClient.UserAgent)
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")

		resp, err := visitorClient.Do(req)
		if err != nil {
			log.Printf("youtube: visitor fetch attempt %d/%d failed: %v", i+1, visitorRetries, err)
			time.Sleep(time.Duration(1<<i) * time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			log.Printf("youtube: visitor fetch attempt %d/%d: status %d", i+1, visitorRetries, resp.StatusCode)
			time.Sleep(time.Duration(1<<i) * time.Second)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("youtube: visitor fetch attempt %d/%d: read error: %v", i+1, visitorRetries, err)
			time.Sleep(time.Duration(1<<i) * time.Second)
			continue
		}
		body = data
		fetched = true
		break
	}

	if !fetched {
		log.Printf("youtube: visitor refresh failed, continuing with cached data")
		return visitorData, currentAPIKey(), nil
	}

	if apiKeyEnv == "" {
		if m := innertubeApiKeyRe.FindSubmatch(body); len(m) > 1 {
			apiKeyMu.Lock()
			apiKey = string(m[1])
			apiKeyMu.Unlock()
		}
	}

	if m := visitorDataRe.FindSubmatch(body); len(m) > 1 {
		visitorData = string(m[1])
	}
	lastFetch = time.Now()

	return visitorData, currentAPIKey(), nil
}
