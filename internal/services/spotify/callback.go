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

type callbackResult struct {
	code string
	err  error
}

type callbackServer struct {
	server *http.Server
	port   int
	result chan callbackResult
	once   sync.Once
}

func newCallbackServer() (*callbackServer, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	c := &callbackServer{
		port:   ln.Addr().(*net.TCPAddr).Port,
		result: make(chan callbackResult, 1),
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
	code := r.URL.Query().Get("code")
	c.once.Do(func() {
		c.result <- callbackResult{code: code, err: nil}
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if runtime.GOOS == "android" {
		fmt.Fprint(w, androidCallbackSuccessPage)
	} else {
		fmt.Fprint(w, callbackSuccessPage)
	}
}

func (c *callbackServer) wait(ctx context.Context) (string, error) {
	select {
	case r := <-c.result:
		return r.code, r.err
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
