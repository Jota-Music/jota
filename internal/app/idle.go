package app

import "github.com/wailsapp/wails/v3/pkg/application"

const idleReloadKey = "idle-reload"

// IdleReload reports whether the app may reload the webview after sitting idle
// and hidden, which is the only way to flush WebKit's internal image caches.
func (a *App) IdleReload() bool {
	var enabled bool
	_ = settings.GetObject(idleReloadKey, &enabled)
	return enabled
}

func (a *App) SetIdleReload(enabled bool) {
	_ = settings.SetObject(idleReloadKey, enabled)
}

// ReloadWindow reloads the webview. The queue and preferences live in
// localStorage, so they survive; only in-memory caches are dropped.
func (a *App) ReloadWindow() {
	if window := application.Get().Window.Current(); window != nil {
		window.Reload()
	}
}
