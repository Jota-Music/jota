package spotify

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type callbackServer struct {
	server *http.Server
	port   int
	// state ties the redirect back to the request that started this flow, so a
	// stray hit on the port cannot deliver a foreign authorization code.
	state  string
	result chan string
	once   sync.Once
}

func newCallbackServer(state string) (*callbackServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	c := &callbackServer{
		port:   ln.Addr().(*net.TCPAddr).Port,
		state:  state,
		result: make(chan string, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/login", c.handle)

	c.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := c.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("spotify callback server error: %v", err)
		}
	}()

	return c, nil
}

func (c *callbackServer) handle(w http.ResponseWriter, r *http.Request) {
	// A mismatched state means this redirect belongs to another request, so it
	// must not consume the slot the real browser is about to fill. Leaving the
	// channel untouched keeps the pending login alive.
	if r.URL.Query().Get("state") != c.state {
		log.Printf("spotify callback: rejected state mismatch")
		http.Error(w, "login mismatch, try again from Jota", http.StatusBadRequest)
		return
	}

	if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
		// The user denied access on Spotify's consent screen. That is a real
		// answer to our request, so it ends the wait instead of spinning until
		// the timeout.
		log.Printf("spotify callback: provider returned error: %s", oauthErr)
		c.once.Do(func() { c.result <- "" })
		fmt.Fprint(w, callbackFailurePage)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "no authorization code", http.StatusBadRequest)
		return
	}
	c.once.Do(func() { c.result <- code })

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if runtime.GOOS == "android" {
		fmt.Fprint(w, androidCallbackSuccessPage)
	} else {
		fmt.Fprint(w, callbackSuccessPage)
	}
}

func (c *callbackServer) wait(ctx context.Context) (string, error) {
	select {
	case code := <-c.result:
		return code, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (c *callbackServer) stop() {
	if c.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = c.server.Shutdown(shutdownCtx)
	}
}

const callbackSuccessPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Jota</title>
<style>
  body{margin:0;font-family:system-ui,-apple-system,sans-serif;background:#0c0a09;color:#e4e7eb;display:grid;place-items:center;min-height:100vh}
  .card{text-align:center;padding:2rem;max-width:30rem}
  h1{color:#22c55e;margin:0 0 0.5rem;font-size:1.5rem}
  p{color:#a1a1aa;margin:0.5rem 0}
</style>
</head>
<body>
  <div class="card">
    <h1>Login complete</h1>
    <p>You can close this tab and return to Jota.</p>
  </div>
  <script>
    setTimeout(function(){ window.close(); }, 800);
  </script>
</body>
</html>`

const callbackFailurePage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Jota</title>
<style>
  body{margin:0;font-family:system-ui,-apple-system,sans-serif;background:#0c0a09;color:#e4e7eb;display:grid;place-items:center;min-height:100vh}
  .card{text-align:center;padding:2rem;max-width:30rem}
  h1{color:#ef4444;margin:0 0 0.5rem;font-size:1.5rem}
  p{color:#a1a1aa;margin:0.5rem 0}
</style>
</head>
<body>
  <div class="card">
    <h1>Login failed</h1>
    <p>Jota could not complete the Spotify login. You can close this tab and try again.</p>
  </div>
</body>
</html>`

// androidCallbackSuccessPage runs inside the Chrome Custom Tab opened for
// OAuth. It deep-links back into the app (scheme registered in Jota's own
// manifest, no Spotify changes); handing control to the app closes the
// custom tab automatically.
const androidCallbackSuccessPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Jota</title>
<style>
  body{margin:0;font-family:system-ui,-apple-system,sans-serif;background:#0c0a09;color:#e4e7eb;display:grid;place-items:center;min-height:100vh}
  .card{text-align:center;padding:2rem;max-width:30rem}
  h1{color:#22c55e;margin:0 0 0.5rem;font-size:1.5rem}
  p{color:#a1a1aa;margin:0.5rem 0}
</style>
</head>
<body>
  <div class="card">
    <h1>Login complete</h1>
    <p>Returning you to Jota...</p>
  </div>
  <script>
    location.replace('intent://callback#Intent;scheme=jota;package=com.jotamusic.jota;end');
    setTimeout(function(){ window.close(); }, 800);
  </script>
</body>
</html>`
