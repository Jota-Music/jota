//go:build !android

package app

import (
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ExportYouTubeCookiesFile exports the stored YouTube session to a file chosen
// by the user. The passphrase stored in settings decides whether the file is
// encrypted. It returns the chosen path, or an empty string when the dialog
// is dismissed.
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

	path, err := application.Get().Dialog.SaveFile().
		SetFilename(filename).
		AddFilter("Text files", "*.txt;*.enc").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}

	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
