package main

import (
	"embed"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"sync/atomic"

	"github.com/Jota-Music/jota/internal/app"
	"github.com/Jota-Music/jota/internal/covers"
	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/services/youtube"

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

const (
	windowMinWidth  = 500
	windowMinHeight = 480
	logMaxBytes     = 2 << 20
)

// rotatingWriter appends to a file and, once it grows past max, moves it aside
// to "<path>.1" (overwriting the previous backup) and starts fresh, so the log
// never grows without bound. One backup file is a deliberate ceiling: bump the
// suffix count if longer history is ever needed.
type rotatingWriter struct {
	path string
	max  int64
	f    *os.File
	size int64
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	if w.size > 0 && w.size+int64(len(p)) > w.max {
		w.rotate()
	}
	n, err := w.f.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingWriter) rotate() {
	_ = w.f.Close()
	_ = os.Rename(w.path, w.path+".1")
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	w.f = f
	w.size = 0
	_ = debug.SetCrashOutput(f, debug.CrashOptions{})
}

// setupLogging mirrors Go logs and fatal crash output to a file next to the
// app's storage, because stderr is lost when the app is launched from a
// desktop entry or an AppImage. Best-effort: no writable dir means stderr only.
func setupLogging() func() {
	path := app.LogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return func() {}
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return func() {}
	}

	w := &rotatingWriter{path: path, max: logMaxBytes, f: f}
	if fi, err := f.Stat(); err == nil {
		w.size = fi.Size()
	}

	log.SetOutput(io.MultiWriter(os.Stderr, w))
	_ = debug.SetCrashOutput(f, debug.CrashOptions{})

	// A native crash (a WebKit abort, for one) kills the process before the
	// deferred cleanup runs, so the marker is what survives to tell the next
	// launch that the previous session died instead of exiting cleanly.
	marker := filepath.Join(filepath.Dir(path), "jota.running")
	if _, err := os.Stat(marker); err == nil {
		log.Printf("previous session ended unexpectedly (crash or forced kill)")
	}
	_ = os.WriteFile(marker, []byte(strconv.Itoa(os.Getpid())), 0o644)

	return func() {
		_ = os.Remove(marker)
		_ = f.Close()
	}
}

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

// ponytail: workaround for Wails v3 macOS clearing the native min size on
// maximise/fullscreen; drop once fixed upstream (still present in beta.23).
func enforceMinSize(w *application.WebviewWindow) {
	if runtime.GOOS != "darwin" || w.IsFullscreen() || w.IsMaximised() {
		return
	}
	if width, height := w.Size(); width < windowMinWidth || height < windowMinHeight {
		w.SetMinSize(windowMinWidth, windowMinHeight)
	}
}

func main() {
	defer setupLogging()()

	log.Printf("jota %s starting", currentVersion)

	a := app.New(currentVersion)

	// Fetch YouTube's visitor token in the background so the first play does not
	// stall while the homepage round-trips.
	youtube.WarmVisitor()

	if os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER") == "" {
		_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "1")
	}
	if os.Getenv("GDK_BACKEND") == "" && os.Getenv("WAYLAND_DISPLAY") != "" {
		_ = os.Setenv("GDK_BACKEND", "x11")
	}

	var window *application.WebviewWindow

	// Wails single-instance on Linux is built on the session D-Bus and aborts
	// the app when there is no bus, so only enable it where one exists.
	var singleInstance *application.SingleInstanceOptions
	if runtime.GOOS != "linux" || os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		singleInstance = &application.SingleInstanceOptions{
			UniqueID: "com.jotamusic.jota",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if window == nil {
					return
				}
				window.Restore()
				window.Show()
				window.Focus()
			},
		}
	}

	wailsApp := application.New(application.Options{
		Name:        "Jota",
		Description: "A self-hosted music streaming app",
		Icon:        icon,
		Services: []application.Service{
			application.NewService(a),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: covers.New().Middleware,
		},
		Linux: application.LinuxOptions{
			ProgramName: "jota", // Linux program name (used in .desktop GTK app id)
		},
		Windows: application.WindowsOptions{
			// WebView2 is Chromium: allow audio to start without a user gesture
			// and keep it running while the window is unfocused or minimised.
			AdditionalBrowserArgs: []string{
				"--autoplay-policy=no-user-gesture-required",
				"--disable-background-timer-throttling",
				"--disable-renderer-backgrounding",
			},
		},
		SingleInstance: singleInstance,
	})

	state := loadWindowState()

	opts := application.WebviewWindowOptions{
		Title:            "Jota",
		Width:            1100,
		Height:           720,
		MinWidth:         windowMinWidth,
		MinHeight:        windowMinHeight,
		Frameless:        true,
		AlwaysOnTop:      state.AlwaysOnTop,
		BackgroundType:   application.BackgroundTypeSolid,
		BackgroundColour: application.NewRGBA(12, 10, 9, 255),
		Mac: application.MacWindow{
			WebviewPreferences: application.MacWebviewPreferences{
				// WKWebView: clear mediaTypesRequiringUserActionForPlayback so
				// audio never waits for a gesture.
				EnableAutoplayWithoutUserAction: application.Enabled,
			},
		},
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

	window = wailsApp.Window.NewWithOptions(opts)

	var restored atomic.Bool
	var quitting atomic.Bool
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		application.Get().Event.Emit("window:always-on-top", state.AlwaysOnTop)
		if restored.Swap(true) || state.Width <= 0 {
			return
		}
		window.SetPosition(state.X, state.Y)
		window.SetSize(max(state.Width, windowMinWidth), max(state.Height, windowMinHeight))
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
		enforceMinSize(window)
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
	log.Printf("jota shutting down")
}
