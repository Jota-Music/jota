//go:build android

package app

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Jota-Music/jota/internal/backup"
	"github.com/Jota-Music/jota/internal/playlists"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ExportBackup stages the serialized playlists, plus the ones the sources
// own, inside the app files dir and hands the file to the system share chooser.
// Resolving the sources is network work while the user may leave the app.
func (a *App) ExportBackup() (string, error) {
	data, err := a.Backup.Export(a.snapshot())
	if err != nil {
		return "", err
	}

	base := application.Android.StoragePath()
	if base == "" {
		return "", errors.New("no Android storage path")
	}
	dir := filepath.Join(base, "playlists")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, backup.BackupFile)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		return "", err
	}

	application.Android.Share(`{"file":` + strconv.Quote(path) + `}`)
	return path, nil
}

// snapshot resolves every external playlist while holding a foreground
// service, so the network calls survive the user leaving the app.
func (a *App) snapshot() []playlists.Snapshot {
	hold()
	defer release()
	return a.externalPlaylists(a.externalSummaries(), true)
}
