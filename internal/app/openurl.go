//go:build !android

package app

import "github.com/wailsapp/wails/v3/pkg/application"

// openExternal opens the URL in the system browser, using the platform opener
// (xdg-open, open, rundll32) that wails already wraps.
func openExternal(url string) error {
	return application.Get().Browser.OpenURL(url)
}
