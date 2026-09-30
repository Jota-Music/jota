//go:build android

package app

import (
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// loginWindow is a shade over the 5-minute OAuth wait in the Spotify service.
// The FGS is normally stopped by SpotifyLoginAndWait's defer; the timer is the
// backstop for the paths that never get there, such as the WebView being
// reloaded and dropping the pending call.
const loginWindow = 6 * time.Minute

// hold keeps the process alive while an OAuth flow is out in the system
// browser. The redirect lands on 127.0.0.1 inside this process, so Android
// freezing or killing the backgrounded app loses the code and the login dies
// silently at the timeout.
func hold() {
	application.Android.StartForegroundService(`{"title":"Jota","text":"Signing in to Spotify…"}`)
	time.AfterFunc(loginWindow, release)
}

func release() {
	application.Android.StopForegroundService()
}
