//go:build !android

package app

import (
	"os"

	"github.com/Jota-Music/jota/internal/backup"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ExportBackup serializes the user's playlists, plus the ones the sources
// own, and asks where to save the file. It returns the chosen path, or an empty
// string when the dialog is dismissed. The dialog comes first so the network
// work only starts once the user picked a destination, and the file carries the
// same name as the sync document: saving it inside the sync folder updates that
// document instead of creating a second file.
func (a *App) ExportBackup() (string, error) {
	path, err := application.Get().Dialog.SaveFile().
		SetFilename(backup.BackupFile).
		AddFilter("JSON files", "*.json").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}

	data, err := a.Backup.Export(a.externalPlaylists(a.externalSummaries(), true))
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
