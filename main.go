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

type windowState struct {
	X, Y, Width, Height int
	Maximised           bool
}

var windowBucket = kv.UseBucket("window")

func loadWindowState() windowState {
	var st windowState
	_ = kv.EnsureStarted()
	_ = windowBucket.GetObject("state", &st)
	return st
}

func saveWindowState(w *application.WebviewWindow) {
	x, y := w.Position()
	width, height := w.Size()
	_ = windowBucket.SetObject("state", windowState{
		X:         x,
		Y:         y,
		Width:     width,
		Height:    height,
		Maximised: w.IsMaximised(),
	})
}

func main() {
	a := app.New()

	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
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
		saveWindowState(window)
	})
	window.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) {
		if !restored.Load() {
			return
		}
		saveWindowState(window)
	})
	window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		if quitting.CompareAndSwap(false, true) {
			wailsApp.Quit()
		}
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
