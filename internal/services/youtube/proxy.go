package youtube

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var proxyOnce sync.Once
var proxyPort int
var proxyMu sync.Mutex
var proxyStreams = map[string]stream{} // videoID -> registered stream

// stream is what the proxy serves for a video: the signed googlevideo URL and
// the content type of the format it holds.
type stream struct {
	url  string
	mime string
}

// proxyClient relays a stream as it comes. It has no total timeout on purpose:
// a body lasts as long as the track, and a deadline that cuts it mid-way looks
// to the player like the end of the file.
var proxyClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: 15 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConnsPerHost:   4,
	},
}

const proxyAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_7_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15"

// StartProxy starts a local HTTP server that relays googlevideo streams under
// simple URLs. WebKit and Chromium handle http://127.0.0.1 better than signed
// googlevideo URLs: no User-Agent gymnastics, no expiring signature, and range
// requests travel as the player sends them.
func StartProxy() int {
	proxyOnce.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/stream/", relay)
		srv := &http.Server{Handler: mux}
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Printf("youtube: proxy failed to listen: %v", err)
			return
		}
		proxyPort = ln.Addr().(*net.TCPAddr).Port
		go srv.Serve(ln)
		log.Printf("youtube: proxy listening on port %d", proxyPort)
	})
	return proxyPort
}

// relay answers one player request with the byte range googlevideo serves. The
// stream is passed through untouched: the status, the byte offsets and the
// headers are googlevideo's own, so a seek starts exactly where the player
// asked and the player can never read the wrong slice of the file. Guessing
// any of that here is what made a seek look like the end of the track.
func relay(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/stream/")
	proxyMu.Lock()
	s, ok := proxyStreams[id]
	proxyMu.Unlock()
	if !ok {
		http.Error(w, "not found", 404)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, s.url, nil)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	req.Header.Set("User-Agent", proxyAgent)
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}

	resp, err := proxyClient.Do(req)
	if err != nil {
		// A seek cancels the request it replaces; that is not a failure.
		if r.Context().Err() != nil {
			return
		}
		log.Printf("youtube: proxy %s err=%v", id, err)
		http.Error(w, "upstream unavailable", 502)
		return
	}
	defer resp.Body.Close()

	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Last-Modified", "ETag"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", s.mime)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil && r.Context().Err() == nil {
		log.Printf("youtube: proxy %s copy err=%v", id, err)
	}
}

// ProxyURL registers a googlevideo URL and returns the local proxy URL for it.
func ProxyURL(videoID, googlevideoURL string) string {
	StartProxy()
	proxyMu.Lock()
	proxyStreams[videoID] = stream{
		url:  unpinned(googlevideoURL),
		mime: streamMime(googlevideoURL),
	}
	proxyMu.Unlock()
	return fmt.Sprintf("http://127.0.0.1:%d/stream/%s", proxyPort, videoID)
}

// unpinned drops the `range` query param YouTube signs into some stream URLs.
// It pins the answer to that slice of the file, so a range request for another
// one is ignored and every seek lands on the wrong bytes.
func unpinned(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || !strings.Contains(u.RawQuery, "range=") {
		return raw
	}
	kept := make([]string, 0, 16)
	for _, part := range strings.Split(u.RawQuery, "&") {
		if !strings.HasPrefix(part, "range=") {
			kept = append(kept, part)
		}
	}
	u.RawQuery = strings.Join(kept, "&")
	return u.String()
}

// streamMime is the type the stream really holds, taken from the URL YouTube
// signed for it. Reporting audio/mp4 for an Opus stream makes the player open
// the wrong demuxer.
func streamMime(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "audio/mp4"
	}
	mime := u.Query().Get("mime")
	if i := strings.IndexByte(mime, ';'); i > 0 {
		mime = mime[:i]
	}
	mime = strings.TrimSpace(mime)
	if !strings.HasPrefix(mime, "audio/") {
		return "audio/mp4"
	}
	return mime
}
