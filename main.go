package main

import (
	"embed"
	"log"
	"os"
	"sync/atomic"

	"jota/server/internal/app"
	"jota/server/internal/kv"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

var currentVersion = "dev"

type windowState struct {
	X, Y, Width, Height int
	Maximised           bool
	AlwaysOnTop         bool
}

var windowBucket = kv.UseBucket("window")

func loadWindowState() windowState {
	var st windowState
	_ = kv.EnsureStarted()
	_ = windowBucket.GetObject("state", &st)
	return st
}

func saveWindowState(w *application.WebviewWindow, alwaysOnTop bool) {
	x, y := w.Position()
	width, height := w.Size()
	_ = windowBucket.SetObject("state", windowState{
		X:           x,
		Y:           y,
		Width:       width,
		Height:      height,
		Maximised:   w.IsMaximised(),
		AlwaysOnTop: alwaysOnTop,
	})
}

func main() {
	a := app.New()

	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
	if os.Getenv("GDK_BACKEND") == "" && os.Getenv("WAYLAND_DISPLAY") != "" {
		_ = os.Setenv("GDK_BACKEND", "x11")
	}

	wailsApp := application.New(application.Options{
		Name:        "Jota",
		Description: "A self-hosted music streaming app",
		Icon:        icon,
		Services: []application.Service{
			application.NewService(a),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Linux: application.LinuxOptions{
			ProgramName: "jota", // Linux program name (used in .desktop GTK app id)
		},
	})

	state := loadWindowState()

	opts := application.WebviewWindowOptions{
		Title:            "Jota",
		Width:            1100,
		Height:           720,
		MinWidth:         320,
		MinHeight:        480,
		Frameless:        true,
		AlwaysOnTop:      state.AlwaysOnTop,
		BackgroundType:   application.BackgroundTypeSolid,
		BackgroundColour: application.NewRGBA(12, 10, 9, 255),
		Linux: application.LinuxWindow{
			Icon:             icon,
			WebviewGpuPolicy: application.WebviewGpuPolicyOnDemand,
		},
		URL: "/",
	}
	if state.Width > 0 {
		opts.Width, opts.Height = state.Width, state.Height
		if state.Maximised {
			opts.StartState = application.WindowStateMaximised
		}
	}

	window := wailsApp.Window.NewWithOptions(opts)

	var restored atomic.Bool
	var quitting atomic.Bool
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		application.Get().Event.Emit("window:always-on-top", state.AlwaysOnTop)
		if restored.Swap(true) || state.Width <= 0 {
			return
		}
		window.SetPosition(state.X, state.Y)
		window.SetSize(state.Width, state.Height)
	})
	window.OnWindowEvent(events.Common.WindowDidMove, func(*application.WindowEvent) {
		if !restored.Load() {
			return
		}
		saveWindowState(window, state.AlwaysOnTop)
	})
	window.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) {
		if !restored.Load() {
			return
		}
		saveWindowState(window, state.AlwaysOnTop)
	})
	window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		if quitting.CompareAndSwap(false, true) {
			wailsApp.Quit()
		}
	})

	application.Get().Event.On("window:always-on-top:toggle", func(*application.CustomEvent) {
		state.AlwaysOnTop = !state.AlwaysOnTop
		window.SetAlwaysOnTop(state.AlwaysOnTop)
		saveWindowState(window, state.AlwaysOnTop)
		application.Get().Event.Emit("window:always-on-top", state.AlwaysOnTop)
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
