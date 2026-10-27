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

// YouTubeBrowserLoginSupported reports whether the desktop in-app window can
// run. Android has no room for an external page inside its single fullscreen
// WebView and no sign-in of its own, so this answers false there: a session
// reaches the phone as an imported cookies.txt or through the sync document.
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

	// WebKitGTK keeps cookies in memory unless a store is set before the window
	// opens, and the store is the only place the HttpOnly half of a session is.
	application.InvokeSync(youtube.PersistWebviewCookies)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("login listener: %w", err)
	}
	defer ln.Close()

	cookies := make(chan string, 4)
	pages := make(chan string, 2)
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
			cookieHeader := r.FormValue("c")
			page := r.FormValue("p")
			if cookieHeader == "" && page == "" {
				body, _ := io.ReadAll(io.LimitReader(r.Body, 8<<20))
				cookieHeader = string(body)
			}
			visitor, version := r.FormValue("v"), r.FormValue("cv")
			log.Printf("youtube: browser login received %d cookie bytes, %d page bytes, visitor %d bytes, client %q",
				len(cookieHeader), len(page), len(visitor), version)
			if cookieHeader != "" {
				select {
				case cookies <- cookieHeader:
				default:
				}
			}
			if page != "" {
				select {
				case pages <- page:
				default:
				}
			}
			if visitor != "" || version != "" {
				youtube.AdoptSession(visitor, version)
				log.Printf("youtube: adopted the session's visitor (%d bytes) and client %q", len(visitor), version)
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, "<!doctype html><title>Jota</title><p>Signed in. You can close this window.</p>")
		}),
	}
	defer server.Close()
	go func() { _ = server.Serve(ln) }()

	// ExecJS is gated on the page loading the Wails runtime, which an external
	// page never does, so inject the collector through WebviewWindowOptions.JS:
	// it runs on every load. Once YouTube's own config says the page is signed
	// in, it fetches the playlists page with the webview's cookies — which
	// include the HttpOnly ones document.cookie cannot see — and hands both
	// values over as a top-level form POST, which sidesteps CORS, PNA and mixed
	// content. The playlists page is read here because the partial cookie set
	// the page can hand over is not enough to sign an innertube request.
	collector := fmt.Sprintf(`(function () {
  var target = %q;
  var sent = false;
  var started = Date.now();
  function cfg(key) {
    try {
      if (!window.ytcfg) return "";
      if (typeof window.ytcfg.get === "function") return window.ytcfg.get(key) || "";
      if (window.ytcfg.data_) return window.ytcfg.data_[key] || "";
    } catch (e) {}
    return "";
  }
  function signedIn() {
    try {
      if (!window.ytcfg) return false;
      if (typeof window.ytcfg.get === "function" &&
          window.ytcfg.get("LOGGED_IN") === true) return true;
      return !!(window.ytcfg.data_ && window.ytcfg.data_.LOGGED_IN === true);
    } catch (e) { return false; }
  }
  function hasSession() {
    return document.cookie.indexOf("SAPISID=") !== -1 ||
      document.cookie.indexOf("__Secure-3PAPISID=") !== -1;
  }
  function send(page) {
    if (sent) return;
    try {
      var form = document.createElement("form");
      form.method = "POST";
      form.action = target;
      function field(name, value) {
        var node = document.createElement("textarea");
        node.name = name;
        node.value = value;
        form.appendChild(node);
      }
      field("c", document.cookie);
      if (page) field("p", page);
      field("v", cfg("VISITOR_DATA") || "");
      field("cv", cfg("INNERTUBE_CONTEXT_CLIENT_VERSION") || cfg("INNERTUBE_CLIENT_VERSION") || "");
      (document.body || document.documentElement).appendChild(form);
      form.submit();
      sent = true;
    } catch (e) {}
  }
  var probing = false;
  function step() {
    if (sent || probing) return;
    if (location.hostname !== "www.youtube.com") return;

    // The fetch itself is the test: it goes with the account's cookies, and the
    // page it answers carries YouTube's own verdict. Neither document.cookie nor
    // ytcfg is reliable for that any more, and waiting for them is what made the
    // window sit there without ever signing the app in.
    probing = true;
    fetch("/feed/playlists", { credentials: "include" })
      .then(function (r) { return r.text(); })
      .then(function (html) {
        if (html.indexOf('"LOGGED_IN":true') !== -1) {
          send(html);
          return;
        }
        // A page that does not claim the session signs nothing; a cookie header
        // that does is still the fallback when the page cannot be read.
        if (signedIn() && hasSession() && Date.now() - started > 45000) send("");
      })
      .catch(function () {})
      .then(function () { probing = false; });
  }
  setInterval(step, 3000);
  step();
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

	importPlaylists := func(page string) error {
		n, err := youtube.ImportMyPlaylists([]byte(page))
		if err != nil {
			log.Printf("youtube: import playlists failed: %v", err)
			return nil
		}
		if n == 0 {
			log.Printf("youtube: the playlists page carried no playlist")
			return nil
		}
		log.Printf("youtube: imported %d playlists", n)
		return nil
	}

	// importSession stores the session held by the sign-in window's own cookie
	// store. document.cookie never carries the HttpOnly cookies, so the beacon
	// alone can only hand over a set that signs requests while staying anonymous.
	// The store is written on the webview's own schedule, so it is polled.
	importSession := func() bool {
		for range 20 {
			raw, err := youtube.WebviewCookies()
			if err == nil {
				if err := youtube.SetCookies(raw); err == nil {
					log.Printf("youtube: session read from the sign-in window's cookie store")
					go func() {
						if err := youtube.Verify(); err != nil {
							log.Printf("youtube: the window's session is not one YouTube accepts: %v", err)
						}
					}()
					return true
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
		log.Printf("youtube: the sign-in window's cookie store stayed empty; the page is all there is")
		return false
	}

	var (
		signedIn bool
		page     string
		pageWait <-chan time.Time
	)

	timeout := time.After(5 * time.Minute)
	for {
		select {
		case raw := <-cookies:
			if raw == "" {
				log.Printf("youtube: the sign-in window posted no cookie header; the page is what counts")
				continue
			}
			if err := youtube.SetCookies(raw); err != nil {
				// A page already fetched with credentials proves the session even
				// when the visible cookies are not the ones that sign requests.
				log.Printf("youtube: browser login cookies rejected (%v); keeping what the page carries", err)
				continue
			}
			signedIn = true
			go func() {
				if err := youtube.Verify(); err != nil {
					log.Printf("youtube: the sign-in window's cookies are not a session YouTube accepts: %v", err)
					return
				}
				log.Printf("youtube: session verified")
			}()
			// Both values usually arrive in the same POST: when the page is
			// already here the login is done and the window closes now.
			if page != "" {
				_ = importPlaylists(page)
				return nil
			}
			// Otherwise wait a bounded moment for the page, so the account's
			// playlists are there when the settings page opens.
			pageWait = time.After(20 * time.Second)
		case html := <-pages:
			if !signedIn || !youtube.SessionFull() {
				// The page was read with the account's own session, so the window's
				// store holds the full cookie set by now. Keeping it before the page
				// is all that separates a real session from the visible half.
				if importSession() {
					signedIn = true
				}
			}
			if !signedIn {
				page = html
				pageWait = time.After(2 * time.Second)
				continue
			}
			_ = importPlaylists(html)
			return nil
		case <-pageWait:
			if page != "" {
				_ = importPlaylists(page)
				log.Printf("youtube: signed in from the page alone; the cookie header did not sign a request")
			}
			return nil
		case <-closed:
			if signedIn {
				// Whatever arrived before the window was closed stays imported.
				return nil
			}
			return errors.New("login window closed")
		case <-timeout:
			if signedIn {
				return nil
			}
			return errors.New("login timed out")
		}
	}
}
