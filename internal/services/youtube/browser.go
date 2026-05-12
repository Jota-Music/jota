package youtube

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

func getStreamURLViaBrowser(youtubeId string) (*StreamInfo, error) {
	browserInstance := os.Getenv("BROWSER_INSTANCE")

	var browser *rod.Browser

	if browserInstance != "" {
		l := launcher.MustNewManaged(browserInstance)
		browser = rod.New().Client(l.MustClient())
	} else {
		u, err := launcher.New().
			Leakless(false).
			Headless(true).
			Set("no-sandbox").
			Set("disable-setuid-sandbox").
			Set("disable-dev-shm-usage").
			Launch()
		if err != nil {
			bin := launcher.NewBrowser().MustGet()
			u, err = launcher.New().
				Bin(bin).
				Leakless(false).
				Headless(true).
				Set("no-sandbox").
				Set("disable-setuid-sandbox").
				Set("disable-dev-shm-usage").
				Launch()
		}
		if err != nil {
			return nil, fmt.Errorf("launch browser: %w", err)
		}
		browser = rod.New().ControlURL(u)
	}

	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("connect browser: %w", err)
	}
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{
		URL: fmt.Sprintf("https://www.youtube.com/watch?v=%s", youtubeId),
	})
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	done := make(chan *StreamInfo, 1)

	page.HijackRequests().MustAdd("*/player*", func(hijack *rod.Hijack) {
		body := string(hijack.Response.Body())
		if strings.Contains(body, "streamingData") {
			playerURL := hijack.Request.URL().String()
			parsed, _ := url.Parse(playerURL)
			if parsed != nil {
				expire := parsed.Query().Get("expire")
				if expire != "" {
					streamURL := extractStreamURL(body)
					if streamURL != "" {
						var expireInt int64
						fmt.Sscanf(expire, "%d", &expireInt)
						done <- &StreamInfo{URL: streamURL, ExpiresAt: expireInt}
					}
				}
			}
		}
		hijack.ContinueRequest(nil)
	})

	go page.HijackRequests().Run()

	select {
	case info := <-done:
		return info, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func extractStreamURL(body string) string {
	parts := strings.Split(body, `"streamingData"`)
	if len(parts) < 2 {
		return ""
	}
	section := parts[1]
	if idx := strings.Index(section, "adaptiveFormats"); idx > 0 {
		segment := section[idx:]
		if urlIdx := strings.Index(segment, `"url":"`); urlIdx > 0 && urlIdx < 500 {
			start := urlIdx + 6
			end := strings.Index(segment[start:], `"`)
			if end > 0 {
				return strings.ReplaceAll(segment[start:start+end], `\u002F`, "/")
			}
		}
	}
	return ""
}