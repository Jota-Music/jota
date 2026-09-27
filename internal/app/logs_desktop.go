//go:build !android

package app

import (
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// SaveLogs asks the user where to save a copy of the log, so a crash report can
// be shared. It returns the chosen path, or an empty string when the dialog is
// dismissed.
func (a *App) SaveLogs() (string, error) {
	content, err := readLogs()
	if err != nil {
		return "", err
	}

	path, err := application.Get().Dialog.SaveFile().
		SetFilename("jota.log").
		AddFilter("Log files", "*.log").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
