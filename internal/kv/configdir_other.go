//go:build !android

package kv

import "os"

// storageBaseDir returns the per-user config directory (e.g. ~/.config on
// Linux). Best-effort: falls back to the user's temp directory so startup
// never aborts just because a platform lacks a home/config dir.
func storageBaseDir() (string, error) {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return dir, nil
	}
	return os.TempDir(), nil
}