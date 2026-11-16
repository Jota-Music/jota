package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// IdleReload reports whether the app may reload the webview after sitting idle
// and hidden, which is the only way to flush WebKit's internal image caches.
func (a *App) IdleReload() bool {
	// Reloading the hidden WebView is the only reliable way to release its
	// internal image caches. Enable this for new installs; a saved user choice
	// still takes precedence.
	if value := readSettings().IdleReload; value != nil {
		return *value
	}
	return true
}

func (a *App) SetIdleReload(enabled bool) {
	editSettings(func(current *deviceSettings) {
		current.IdleReload = &enabled
	})
}

// ReloadWindow reloads the webview. The queue and preferences live in
// localStorage, so they survive; only in-memory caches are dropped.
func (a *App) ReloadWindow() {
	if window := application.Get().Window.Current(); window != nil {
		window.Reload()
	}
}

// RamMB reports the memory the app costs: this process plus its WebKit helpers
// (renderer, network, GPU), so the frontend can flush WebKit's internal image
// caches when they are what is growing.
//
// The number is the proportional set size, not RSS. RSS counts every shared
// library once per process, and a WebKit app maps the same few hundred
// megabytes of engine into each helper: summing it answers with roughly twice
// what the app holds. PSS splits a shared page between the processes that map
// it, which is the figure a memory monitor reports as the app's own. Zero means
// the platform exposes no /proc and the frontend falls back to its deadline.
func (a *App) RamMB() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	self := os.Getpid()
	ppid := make(map[int]int, len(entries))
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join("/proc", e.Name(), "status"))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			if !strings.HasPrefix(line, "PPid:") {
				continue
			}
			fields := strings.Fields(strings.TrimPrefix(line, "PPid:"))
			if len(fields) > 0 {
				ppid[pid], _ = strconv.Atoi(fields[0])
			}
			break
		}
	}
	kb := 0
	for pid := range ppid {
		p := pid
		for i := 0; i < 16 && p != 0 && p != self; i++ {
			p = ppid[p]
		}
		if p == self {
			kb += pssKB(pid)
		}
	}
	return kb / 1024
}

// pssKB is one process's proportional set size, read from the rollup the kernel
// keeps for exactly this question: summing its mappings would mean parsing
// smaps, which is megabytes of text for a process holding a few hundred.
func pssKB(pid int) int {
	b, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "smaps_rollup"))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "Pss:") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "Pss:"))
		if len(fields) > 0 {
			kb, _ := strconv.Atoi(fields[0])
			return kb
		}
	}
	return 0
}
