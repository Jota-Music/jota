package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

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

// RamMB reports the resident memory of the app plus its WebKit helper
// processes (renderer, network, GPU), so the frontend can flush WebKit's
// internal image caches exactly when they are what is growing. Zero means the
// platform exposes no /proc and the frontend falls back to its deadline.
func (a *App) RamMB() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	type pstat struct {
		ppid int
		rss  int
	}
	self := os.Getpid()
	stats := make(map[int]pstat)
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join("/proc", e.Name(), "status"))
		if err != nil {
			continue
		}
		var s pstat
		for _, line := range strings.Split(string(b), "\n") {
			switch {
			case strings.HasPrefix(line, "PPid:"):
				fields := strings.Fields(strings.TrimPrefix(line, "PPid:"))
				if len(fields) > 0 {
					s.ppid, _ = strconv.Atoi(fields[0])
				}
			case strings.HasPrefix(line, "VmRSS:"):
				fields := strings.Fields(strings.TrimPrefix(line, "VmRSS:"))
				if len(fields) > 0 {
					s.rss, _ = strconv.Atoi(fields[0])
				}
			}
		}
		stats[pid] = s
	}
	kb := 0
	for pid, s := range stats {
		p := pid
		for i := 0; i < 16 && p != 0 && p != self; i++ {
			p = stats[p].ppid
		}
		if p == self {
			kb += s.rss
		}
	}
	return kb / 1024
}
