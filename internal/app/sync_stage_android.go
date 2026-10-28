//go:build android

package app

import (
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// syncStageDir is where the document crosses between the native side and the
// backend. It is the app files dir Java writes to as well (getFilesDir()/sync),
// so a document moves as a file and never as a string over the bridge.
func syncStageDir() string {
	base := application.Android.StoragePath()
	if base == "" {
		return ""
	}
	dir := filepath.Join(base, "sync")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	return dir
}
