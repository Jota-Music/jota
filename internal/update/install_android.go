//go:build android

package update

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// updatesDir stages the APK in the app's private files dir so the Java
// FileProvider can expose it to the system package installer.
func updatesDir() (string, error) {
	base := application.Android.StoragePath()
	if base == "" {
		return "", fmt.Errorf("update: no Android storage path")
	}
	dir := filepath.Join(base, "updates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func targetDir() (string, error) { return updatesDir() }

func install(staged string) error {
	dir, err := updatesDir()
	if err != nil {
		return err
	}
	// jota.apk (fixed name) is what WailsBridge.installApk hands to the
	// system installer via the "updates" FileProvider path.
	if err := os.Rename(staged, filepath.Join(dir, "jota.apk")); err != nil {
		return fmt.Errorf("update: stage APK: %w", err)
	}
	return nil
}
