package app

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// LogPath is the file the app mirrors its log and fatal crash output to. stderr
// is lost when the app is launched from a desktop entry or an AppImage, so this
// is the file a user shares when reporting a crash.
func LogPath() string {
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "jota", "jota.log")
}

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

// readLogs returns the rotated log followed by the current one, so the copy
// reads chronologically.
func readLogs() ([]byte, error) {
	return readLogsAt(LogPath())
}

func readLogsAt(path string) ([]byte, error) {
	var buf bytes.Buffer
	for _, p := range []string{path + ".1", path} {
		content, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		buf.Write(content)
	}
	return buf.Bytes(), nil
}
