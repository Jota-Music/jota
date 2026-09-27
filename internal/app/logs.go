package app

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/Jota-Music/jota/internal/kv"
)

// LogPath is the file the app mirrors its log and fatal crash output to, next to
// the app's storage. stderr is lost when the app is launched from a desktop
// entry or an AppImage, so this is the file a user shares when reporting a crash.
func LogPath() string {
	dir, _ := kv.Dir()
	return filepath.Join(dir, "jota", "jota.log")
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
