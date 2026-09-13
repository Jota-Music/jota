//go:build android

package app

import "github.com/wailsapp/wails/v3/pkg/application"

// openExternal opens the URL in the system browser (Intent.ACTION_VIEW) so
// Spotify OAuth runs outside the app's WebView, where bot-detection
// challenges fail.
func openExternal(url string) error {
	application.Android.OpenURL(url)
	return nil
}
