package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Jota-Music/jota/internal/app"
	"github.com/Jota-Music/jota/internal/covers"
	"github.com/Jota-Music/jota/internal/kv"
	"github.com/Jota-Music/jota/internal/services/youtube"
	"github.com/Jota-Music/jota/internal/single"

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

// windowPath is where the window geometry lives: a tiny file next to the log,
// not a Badger key, so building the window never waits for the database. Set by
// setupLogging, before the window is built.
var windowPath string

const (
	windowMinWidth  = 500
	windowMinHeight = 480
	logMaxBytes     = 2 << 20

	// startupBudget is how long the process may go without a window before it
	// dumps every goroutine. Nothing on the startup path should block, and a
	// block is silent by definition: no window, no log line.
	startupBudget = 20 * time.Second

	// markerStarted and markerReady are the states recorded in the crash marker.
	// A session that dies in "starting" never opened its window.
	markerStarted = "starting"
	markerReady   = "ready"
)

// runningMarker is the crash marker path, set by setupLogging.
var runningMarker string

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
	windowPath = filepath.Join(filepath.Dir(path), "window.json")
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
	// launch how the previous session died. claimMarker owns it from here.
	runningMarker = filepath.Join(filepath.Dir(path), "jota.running")

	return func() {
		_ = f.Close()
	}
}

// claimMarker reports how the previous session left the marker, then records this
// one. The state is the whole point: a session that died while "starting" is a
// launch that never opened a window, which is the failure this app used to hide.
func claimMarker() {
	if runningMarker == "" {
		return
	}
	switch previousState() {
	case markerReady:
		log.Printf("previous session ended unexpectedly (crash or forced kill)")
	case markerStarted:
		log.Printf("previous session ended unexpectedly (crash or forced kill), window never appeared")
	}
	markRunning(markerStarted)
}

// markRunning moves the marker to state.
func markRunning(state string) {
	if runningMarker == "" {
		return
	}
	_ = os.WriteFile(runningMarker, []byte(strconv.Itoa(os.Getpid())+" "+state), 0o644)
}

// previousState is the state left in the marker, or "" when there is none. An
// empty string from a clean exit and from a missing marker are the same answer:
// nothing worth reporting.
func previousState() string {
	data, err := os.ReadFile(runningMarker)
	if err != nil {
		return ""
	}
	_, state, _ := strings.Cut(strings.TrimSpace(string(data)), " ")
	return state
}

func releaseMarker() {
	if runningMarker != "" {
		_ = os.Remove(runningMarker)
	}
}

// watchStartup names the stuck call in the log if no window is up inside the
// budget. Without it a startup block is invisible: the process lives, nothing is
// printed, and the only symptom is a window that never appears.
func watchStartup() *time.Timer {
	return time.AfterFunc(startupBudget, func() {
		var dump bytes.Buffer
		_ = pprof.Lookup("goroutine").WriteTo(&dump, 1)
		log.Printf("startup: no window after %s; goroutines:\n%s", startupBudget, dump.String())
	})
}

func loadWindowState() windowState {
	var st windowState
	data, err := os.ReadFile(windowPath)
	if err != nil {
		return st
	}
	_ = json.Unmarshal(data, &st)
	return st
}

// migrateWindowState copies the geometry older builds kept in Badger into the
// file, once. It runs off the launch path on purpose: opening the database there
// is exactly what moving the state out of it avoided, so this session keeps the
// defaults it started with and the next one gets the geometry back.
func migrateWindowState() {
	if _, err := os.Stat(windowPath); err == nil {
		return
	}
	var st windowState
	if err := kv.EnsureStarted(); err != nil {
		return
	}
	if err := kv.UseBucket("window").GetObject("state", &st); err != nil {
		return
	}

	windowSaveMu.Lock()
	defer windowSaveMu.Unlock()
	// A window that already saved its own geometry is newer than the old key.
	if _, err := os.Stat(windowPath); err == nil {
		return
	}
	writeWindowStateFile(st)
}

// writeWindowStateFile publishes the geometry with temp + rename, so a kill
// mid-write cannot leave a clipped one behind. Every caller holds windowSaveMu,
// so the single temp name cannot be shared by two writers.
func writeWindowStateFile(st windowState) {
	data, err := json.Marshal(st)
	if err != nil {
		return
	}
	tmp := windowPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	if err := os.Rename(tmp, windowPath); err != nil {
		_ = os.Remove(tmp)
	}
}

// clampWindowState keeps the restored window usable. Save-on-resize stores the
// geometry as reported by the window, so a maximize/restore cycle can leave a
// "normal" window the size of the whole monitor (dragging and maximize/restore
// then appear dead). Drop a geometry that fills a screen's work area and cap
// any oversized one to the screen it sits on.
func clampWindowState(st *windowState, screens []*application.Screen) *application.Screen {
	if st.Width <= 0 || len(screens) == 0 {
		return nil
	}
	screen := screens[0]
	for _, s := range screens {
		wa := s.WorkArea
		if st.X >= wa.X && st.X < wa.X+wa.Width && st.Y >= wa.Y && st.Y < wa.Y+wa.Height {
			screen = s
			break
		}
	}
	wa := screen.WorkArea
	if st.Width >= wa.Width && st.Height >= wa.Height {
		st.Width, st.Height = 0, 0
		return screen
	}
	st.Width = min(st.Width, wa.Width)
	st.Height = min(st.Height, wa.Height)

	// Size alone is not enough: a position saved on a monitor that is gone, or on
	// a layout that shrank, restores the window off-screen with no way to drag it
	// back. Pull it inside the work area.
	st.X = min(max(st.X, wa.X), wa.X+wa.Width-st.Width)
	st.Y = min(max(st.Y, wa.Y), wa.Y+wa.Height-st.Height)
	return screen
}

// Moving or resizing the window floods WindowDidMove/Resize events at every
// pointer move, and each state write churns the Badger value log. Throttle the
// writes to one per interval and keep the freshest pending state; the last one
// is flushed at WindowClosing and again after the app exits.
const windowSaveInterval = 400 * time.Millisecond

// windowSaveState is the state to persist. Only the event handlers and the pin
// button write it, from a live window; the close hook only flushes it, so nothing
// reads a window Wails is already tearing down. The pin lives here rather than in
// the startup state, so the startup value is read-only once the window exists.
var (
	windowSaveMu    sync.Mutex
	windowLastSave  time.Time
	windowSaveState windowState
	windowSaveDirty bool
)

// snapshot reads the live window. Call it from an event handler only.
func snapshot(w *application.WebviewWindow) windowState {
	x, y := w.Position()
	width, height := w.Size()
	return windowState{
		X:         x,
		Y:         y,
		Width:     width,
		Height:    height,
		Maximised: w.IsMaximised(),
	}
}

func saveWindowState(st windowState) {
	windowSaveMu.Lock()
	st.AlwaysOnTop = windowSaveState.AlwaysOnTop
	windowSaveState = st
	windowSaveDirty = true
	due := time.Since(windowLastSave) >= windowSaveInterval
	if due {
		windowLastSave = time.Now()
	}
	windowSaveMu.Unlock()
	if due {
		flushWindowState()
	}
}

// flushWindowState persists the cache without touching the window. A cache with
// no geometry is dropped: the window can report 0x0 before it is mapped, and
// saving that would restore a window with no size.
func flushWindowState() {
	windowSaveMu.Lock()
	defer windowSaveMu.Unlock()
	if !windowSaveDirty || windowSaveState.Width <= 0 {
		return
	}
	windowSaveDirty = false
	writeWindowStateFile(windowSaveState)
}

// getAlwaysOnTop reads the pinned flag from the cache, the single source of truth
// after startup.
func getAlwaysOnTop() bool {
	windowSaveMu.Lock()
	defer windowSaveMu.Unlock()
	return windowSaveState.AlwaysOnTop
}

// setAlwaysOnTop records the pinned flag and persists it in the same pass, so a
// relaunch restores the pin without waiting for a move or resize.
func setAlwaysOnTop(on bool) {
	windowSaveMu.Lock()
	windowSaveState.AlwaysOnTop = on
	windowSaveDirty = true
	windowSaveMu.Unlock()
	flushWindowState()
}

// shedMemory returns the Go heap to the OS once the window is hidden. The
// webview owns most of the memory, but while the app sits in the background
// there is no reason for Go to keep pages it no longer needs. Debounced so a
// minimise/hide burst only triggers one collection.
var (
	shedMu   sync.Mutex
	shedLast time.Time
)

func shedMemory() {
	shedMu.Lock()
	if time.Since(shedLast) < time.Minute {
		shedMu.Unlock()
		return
	}
	shedLast = time.Now()
	shedMu.Unlock()
	go debug.FreeOSMemory()
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

// startProfiling records a CPU profile for the whole run and a heap profile on
// exit when JOTA_PROFILE_DIR is set. Opt-in, so the default build pays nothing,
// and starting with the process is the only way to capture startup work such as
// opening Badger or building the window.
func startProfiling() func() {
	dir := os.Getenv("JOTA_PROFILE_DIR")
	if dir == "" {
		return func() {}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("profile: %v", err)
		return func() {}
	}
	cpu, err := os.Create(filepath.Join(dir, "cpu.pprof"))
	if err != nil {
		log.Printf("profile: %v", err)
		return func() {}
	}
	if err := pprof.StartCPUProfile(cpu); err != nil {
		log.Printf("profile: %v", err)
		_ = cpu.Close()
		return func() {}
	}
	log.Printf("profile: cpu -> %s", cpu.Name())
	return func() {
		pprof.StopCPUProfile()
		_ = cpu.Close()
		heap, err := os.Create(filepath.Join(dir, "heap.pprof"))
		if err != nil {
			log.Printf("profile: %v", err)
			return
		}
		defer heap.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(heap); err != nil {
			log.Printf("profile: %v", err)
		}
	}
}

func main() {
	defer setupLogging()()
	defer startProfiling()()

	// The webview owns most of the memory; keep Go's heap small and bounded so
	// it stops competing with WebKit instead of ballooning to 2x live data.
	// GOMEMLIMIT is a soft limit: Go just collects harder, so 128MB cannot OOM
	// the app. Explicit GOGC/GOMEMLIMIT env vars still win for tuning.
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(50)
	}
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(128 << 20)
	}

	startedAt := time.Now()

	// The single instance is decided before anything else starts, so a second
	// launch costs a socket round trip and nothing more.
	var windowPtr atomic.Pointer[application.WebviewWindow]
	focus := func() {
		w := windowPtr.Load()
		if w == nil {
			return
		}
		w.Restore()
		w.Show()
		w.Focus()
	}
	if runtime.GOOS == "linux" {
		guard, err := single.Acquire(focus)
		if errors.Is(err, single.ErrHandedOff) {
			log.Printf("jota: already running, asked it to show its window")
			return
		}
		if err != nil {
			log.Printf("single: no session lock, starting anyway: %v", err)
		} else {
			defer guard.Close()
		}
	}

	claimMarker()
	defer releaseMarker()

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

	// Wails' single instance is built on the session D-Bus and its handshake has
	// no timeout, so it is only used where Linux's own guard (internal/single)
	// does not run.
	var singleInstance *application.SingleInstanceOptions
	if runtime.GOOS != "linux" {
		singleInstance = &application.SingleInstanceOptions{
			UniqueID: "com.jotamusic.jota",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				focus()
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
	windowSaveState = state

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
		screen := clampWindowState(&state, application.Get().Screen.GetAll())
		opts.Width, opts.Height = state.Width, state.Height
		// Wails centres a window whose InitialPosition is left unset and the ready
		// event then moves it, which is the jump the user sees. Linux takes X/Y
		// relative to a screen's work area, so the screen holding the saved point
		// is handed over too; the other platforms keep the restore they had.
		if runtime.GOOS == "linux" && screen != nil {
			opts.Screen = screen
			opts.InitialPosition = application.WindowXY
			opts.X = state.X - screen.WorkArea.X
			opts.Y = state.Y - screen.WorkArea.Y
		}
		if state.Maximised {
			opts.StartState = application.WindowStateMaximised
		}
	}

	window = wailsApp.Window.NewWithOptions(opts)
	windowPtr.Store(window)

	watchdog := watchStartup()

	var restored atomic.Bool
	var quitting atomic.Bool
	window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		watchdog.Stop()
		markRunning(markerReady)
		// One number, always in the log: how long the launch took. Comparing it
		// across releases needs no profiler, and a regression shows up as a
		// bigger number instead of a feeling.
		log.Printf("window ready in %s", time.Since(startedAt).Round(time.Millisecond))
		// In the background: opening Badger replays the value log, and neither the
		// service startup nor the first render should wait for that.
		go func() {
			kv.Start()
			migrateWindowState()
		}()
		application.Get().Event.Emit("window:always-on-top", getAlwaysOnTop())
		// The geometry is already the one the window was created with, so there is
		// nothing to move here. Marking it restored lets the move/resize events start
		// saving.
		restored.Store(true)
	})
	window.OnWindowEvent(events.Common.WindowDidMove, func(*application.WindowEvent) {
		if !restored.Load() {
			return
		}
		saveWindowState(snapshot(window))
	})
	window.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) {
		if !restored.Load() {
			return
		}
		enforceMinSize(window)
		saveWindowState(snapshot(window))
	})
	window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		if quitting.CompareAndSwap(false, true) {
			flushWindowState()
			wailsApp.Quit()
		}
	})
	window.OnWindowEvent(events.Common.WindowMinimise, func(*application.WindowEvent) {
		shedMemory()
	})
	window.OnWindowEvent(events.Common.WindowHide, func(*application.WindowEvent) {
		shedMemory()
	})

	application.Get().Event.On("window:always-on-top:toggle", func(*application.CustomEvent) {
		on := !getAlwaysOnTop()
		window.SetAlwaysOnTop(on)
		setAlwaysOnTop(on)
		application.Get().Event.Emit("window:always-on-top", on)
	})

	// The window is created with the saved state before the frontend even loads,
	// so the runtime-ready broadcast can beat the pin button's subscription and
	// the two would disagree. Answer on demand instead of racing the mount.
	application.Get().Event.On("window:always-on-top:get", func(*application.CustomEvent) {
		application.Get().Event.Emit("window:always-on-top", getAlwaysOnTop())
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
	flushWindowState()
	log.Printf("jota shutting down")
}
