package app

import "github.com/Jota-Music/jota/internal/clipboard"

// ClipboardText reads the desktop clipboard off the UI thread. A platform with no
// asynchronous read reports an error, and the page falls back to the runtime's
// own clipboard, which does not block a GTK loop there.
func (a *App) ClipboardText() (string, error) {
	return clipboard.Text()
}
