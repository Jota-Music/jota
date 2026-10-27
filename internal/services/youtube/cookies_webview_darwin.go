//go:build darwin

package youtube

import (
	"os"
	"path/filepath"
	"time"
)

// WebviewCookies returns the Cookie header held by the in-app sign-in window's
// own store. document.cookie never carries the HttpOnly half of a Google
// session, so the store is the only way the window can hand over a real one.
func WebviewCookies() (string, error) {
	path := webviewJar()
	if path == "" {
		return "", errWebviewJar
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errWebviewJar
	}
	rows, err := parseBinaryCookies(data)
	if err != nil || len(rows) == 0 {
		return "", errWebviewJar
	}
	header, _ := cookieHeader(rows, nil)
	if header == "" {
		return "", errWebviewJar
	}
	return header, nil
}

// webviewJar is WKWebView's cookie store. WebKit names its data store after the
// bundle identifier the app is packaged with, and falls back to whichever store
// moved last, which is what a dev build without a bundle leaves behind.
func webviewJar() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	root := filepath.Join(home, "Library", "WebKit")
	if path := cookieFile(filepath.Join(root, "com.jotamusic.jota", "WebsiteData", "Cookies", "Cookies.binarycookies")); path != "" {
		return path
	}
	return newestJar(root)
}

func newestJar(root string) string {
	matches, _ := filepath.Glob(filepath.Join(root, "*", "WebsiteData", "Cookies", "Cookies.binarycookies"))
	newest := ""
	var newestAt time.Time
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil || !info.ModTime().After(newestAt) {
			continue
		}
		newest, newestAt = path, info.ModTime()
	}
	return newest
}
