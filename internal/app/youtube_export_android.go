//go:build android

package app

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ExportYouTubeCookiesFile exports the stored YouTube session to the app storage
// and hands it to the system share chooser. The passphrase stored in settings
// decides whether the file is encrypted.
func (a *App) ExportYouTubeCookiesFile() (string, error) {
	pass := a.SessionPass()
	data, err := a.exportYouTubeCookies(pass)
	if err != nil {
		return "", err
	}

	filename := "youtube-cookies.txt"
	if pass != "" {
		filename = "youtube-cookies.enc"
	}

	base := application.Android.StoragePath()
	if base == "" {
		return "", errors.New("no Android storage path")
	}
	dir := filepath.Join(base, "youtube")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		return "", err
	}

	application.Android.Share(`{"file":` + strconv.Quote(path) + `}`)
	return path, nil
}
