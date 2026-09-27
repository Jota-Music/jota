//go:build android

package app

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// SaveLogs has no counterpart on Android: a save dialog cannot hand back a
// filesystem path there. The log is staged inside the app files dir (exposed to
// the system by the "logs" FileProvider path) and handed to the share chooser
// as a file attachment instead.
func (a *App) SaveLogs() (string, error) {
	content, err := readLogs()
	if err != nil {
		return "", err
	}

	base := application.Android.StoragePath()
	if base == "" {
		return "", errors.New("no Android storage path")
	}
	dir := filepath.Join(base, "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, "jota.log")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", err
	}

	application.Android.Share(`{"file":` + strconv.Quote(path) + `}`)
	return path, nil
}
