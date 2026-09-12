//go:build android

package kv

import (
	"errors"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// storageBaseDir returns the app's private internal files directory
// (activity.getFilesDir()), which always exists on Android. Falls back to the
// per-app temp dir if the bridge reports an empty path.
func storageBaseDir() (string, error) {
	if p := application.Android.StoragePath(); p != "" {
		return p, nil
	}
	if os.Getenv("TMPDIR") != "" {
		return os.TempDir(), nil
	}
	return "", errors.New("android storage path unavailable")
}