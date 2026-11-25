package app

import (
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// A WKWebView draws in a helper process, and macOS kills that helper under
// memory pressure (macOS 26 does it while the app is drawing) or when the Mac
// sleeps. The window is then left white, this process is alive, and the page
// that would have logged something is the one that died.
//
// Wails already reloads the webview when its navigation delegate reports the
// termination, but a reload is a request to the same item and it can leave the
// window blank for good. The way back is a navigation, and the page is asked to
// prove it is alive before taking it: nothing in the page can report a renderer
// that no longer exists.
const (
	// beatWait is how long the page has to answer a probe. It is one bridge call
	// from the main thread, so anything slower is no answer.
	beatWait = 5 * time.Second
	// beatFallback is how long Wails' own reload gets after a termination before
	// this takes over. The reload keeps the route; a navigation lands on the
	// shelf.
	beatFallback = 3 * time.Second
	// reviveGap keeps a renderer that keeps dying from becoming a navigation
	// loop. A recovery inside the gap is retried once it passes, not dropped.
	reviveGap = 30 * time.Second
)

type rendererState struct {
	sync.Mutex
	seen    bool
	beatAt  time.Time
	revived time.Time
	closing bool
}

// MainWindowName identifies the app's own window. The YouTube sign-in opens a
// second one, and neither the recovery nor the idle reload may touch it.
const MainWindowName = "main"

var renderer rendererState

// Beat is the page answering a probe. It reports nothing by itself: the answer
// is the beat.
func (a *App) Beat() {
	renderer.Lock()
	renderer.beat(time.Now())
	renderer.Unlock()
}

// watchRenderer arms the recovery. macOS only: the event is the WKWebView
// delegate, and the other platforms discard a frozen renderer by themselves.
func (a *App) watchRenderer() {
	if runtime.GOOS != "darwin" {
		return
	}
	window := mainWindow()
	if window == nil {
		log.Printf("webview: no window named %q, renderer recovery is off", MainWindowName)
		return
	}
	window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		renderer.Lock()
		renderer.closing = true
		renderer.Unlock()
	})
	window.OnWindowEvent(events.Mac.WebViewWebContentProcessDidTerminate, func(*application.WindowEvent) {
		log.Printf("webview: content process terminated")
		asked := time.Now()
		time.AfterFunc(beatFallback, func() {
			revive(window, asked)
		})
	})
	// The other way in: the renderer died while the window was hidden, so no
	// delegate fired and the user only finds white when coming back.
	for _, event := range []events.WindowEventType{events.Common.WindowShow, events.Mac.WindowDidBecomeKey} {
		window.OnWindowEvent(event, func(*application.WindowEvent) {
			ask(window)
		})
	}
}

// mainWindow returns the app's own window, or nil before it exists. It is
// looked up by name because the YouTube sign-in opens a second one, and that
// one must never be reloaded or navigated by the recovery.
func mainWindow() *application.WebviewWindow {
	window, ok := application.Get().Window.GetByName(MainWindowName)
	if !ok {
		return nil
	}
	webview, _ := window.(*application.WebviewWindow)
	return webview
}

// ask has the page prove it is alive. A dead renderer executes nothing, so
// silence past the wait is the answer. A page that has never beaten is still
// booting and is left alone.
func ask(window *application.WebviewWindow) {
	renderer.Lock()
	seen := renderer.seen
	asked := time.Now()
	renderer.Unlock()
	if !seen {
		return
	}
	window.ExecJS("window.__jotaBeat?.()")
	time.AfterFunc(beatWait, func() {
		revive(window, asked)
	})
}

// revive loads the page again when it has not answered since asked. The window
// is checked again: SetURL has no destroyed check of its own, and a timer
// outlives it.
func revive(window *application.WebviewWindow, asked time.Time) {
	renderer.Lock()
	wait, open := renderer.when(time.Now())
	if open && wait > 0 {
		renderer.Unlock()
		time.AfterFunc(wait, func() { revive(window, asked) })
		return
	}
	if !open || !renderer.silent(asked) {
		renderer.Unlock()
		return
	}
	renderer.revived = time.Now()
	renderer.Unlock()
	if mainWindow() != window {
		return
	}
	log.Printf("webview: the page did not answer, loading it again")
	window.SetURL("/")
}

func (r *rendererState) beat(now time.Time) {
	r.seen = true
	r.beatAt = now
}

// silent reports whether the page has not beaten since since. A closing window
// is never revived: the process is on its way out.
func (r *rendererState) silent(since time.Time) bool {
	return !r.closing && r.beatAt.Before(since)
}

// when reports how long a recovery must still wait and whether it may ever run.
func (r *rendererState) when(now time.Time) (time.Duration, bool) {
	if r.closing {
		return 0, false
	}
	if remain := reviveGap - now.Sub(r.revived); remain > 0 {
		return remain, true
	}
	return 0, true
}
