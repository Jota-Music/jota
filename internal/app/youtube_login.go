package app

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/Jota-Music/jota/internal/services/youtube"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// YouTubeBrowserLoginSupported reports whether the in-app browser login can run.
// On Android the WebView is a single fullscreen view owned by the Activity and
// cannot load an external URL, so only paste and cookies.txt import work there.
func (a *App) YouTubeBrowserLoginSupported() bool {
	return runtime.GOOS != "android"
}

// YouTubeBrowserLogin opens a window on youtube.com so the user can sign in
// normally; the page beacons its document.cookie back to a local listener, which
// is stored for innertube requests. Blocks until signed in, the window is
// closed, or it times out.
func (a *App) YouTubeBrowserLogin() error {
	if !a.YouTubeBrowserLoginSupported() {
		return errors.New("in-app browser login is not available on this platform")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("login listener: %w", err)
	}
	defer ln.Close()

	cookies := make(chan string, 8)
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.Header().Set("Access-Control-Allow-Private-Network", "true")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			_ = r.ParseForm()
			value := r.FormValue("c")
			if value == "" {
				body, _ := io.ReadAll(io.LimitReader(r.Body, 64<<10))
				value = string(body)
			}
			log.Printf("youtube: browser login received %d bytes", len(value))
			if value != "" {
				select {
				case cookies <- value:
				default:
				}
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, "<!doctype html><title>Jota</title><p>Signed in. You can close this window.</p>")
		}),
	}
	defer server.Close()
	go func() { _ = server.Serve(ln) }()

	// ExecJS is gated on the page loading the Wails runtime, which an external
	// page never does, so inject the collector through WebviewWindowOptions.JS:
	// it runs on every load. Once SAPISID shows up it hands document.cookie over
	// as a top-level form POST, which sidesteps CORS, PNA and mixed content.
	collector := fmt.Sprintf(`(function () {
  var target = %q;
  function send() {
    try {
      if (document.cookie.indexOf("SAPISID=") === -1 &&
          document.cookie.indexOf("__Secure-3PAPISID=") === -1) return;
      var form = document.createElement("form");
      form.method = "POST";
      form.action = target;
      var field = document.createElement("textarea");
      field.name = "c";
      field.value = document.cookie;
      form.appendChild(field);
      (document.body || document.documentElement).appendChild(form);
      form.submit();
    } catch (e) {}
  }
  setInterval(send, 1500);
  send();
})();`, fmt.Sprintf("http://127.0.0.1:%d/", ln.Addr().(*net.TCPAddr).Port))

	window := application.Get().Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Sign in to YouTube",
		URL:    "https://www.youtube.com/",
		Width:  520,
		Height: 760,
		JS:     collector,
	})
	defer window.Close()

	closed := make(chan struct{})
	window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		select {
		case <-closed:
		default:
			close(closed)
		}
	})

	timeout := time.After(3 * time.Minute)
	for {
		select {
		case raw := <-cookies:
			if err := youtube.SetCookies(raw); err != nil {
				log.Printf("youtube: browser login cookies rejected: %v", err)
				continue
			}
			return nil
		case <-closed:
			return errors.New("login window closed")
		case <-timeout:
			return errors.New("login timed out")
		}
	}
}
