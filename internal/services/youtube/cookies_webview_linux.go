//go:build linux && !android

package youtube

import (
	"os"
	"path/filepath"
)

// WebviewCookies returns the Cookie header held by the in-app sign-in window's
// own store. document.cookie never carries the HttpOnly half of a Google
// session, so the store is the only way the window can hand over a real one.
func WebviewCookies() (string, error) {
	path := webviewJar()
	if path == "" {
		return "", errWebviewJar
	}
	rows, _, err := readJar(path)
	if err != nil {
		return "", err
	}
	header, _ := cookieHeader(rows, nil)
	if header == "" {
		return "", errWebviewJar
	}
	return header, nil
}

// webviewDataDir is the data directory the sign-in window runs on: the default
// network session, named after the program name Wails pins for the app.
func webviewDataDir() string {
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		data = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(data, "jota")
}

// cookieStore is where WebKitGTK writes the window's jar.
func cookieStore() string {
	dir := webviewDataDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "cookies.sqlite")
}

// webviewJar is that store on disk. An older build named the directory with a
// capital J, so both spellings are read.
func webviewJar() string {
	store := cookieStore()
	if store == "" {
		return ""
	}
	return cookieFile(store, filepath.Join(filepath.Dir(store), "Jota", "cookies.sqlite"))
}
