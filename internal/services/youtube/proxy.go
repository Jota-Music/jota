package youtube

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var proxyOnce sync.Once
var proxyPort int
var proxyMu sync.Mutex
var googleMu sync.Mutex                // serialize requests to googlevideo to avoid rate limiting
var proxyStreams = map[string]string{} // videoID -> googlevideo URL

// StartProxy starts a local HTTP server that serves googlevideo streams under
// simple URLs. WebKit handles http://localhost better than signed googlevideo
// URLs: it buffers properly and range requests work without the client's
// User-Agent gymnastics.
func StartProxy() int {
	proxyOnce.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/stream/", func(w http.ResponseWriter, r *http.Request) {
			googleMu.Lock()

			id := strings.TrimPrefix(r.URL.Path, "/stream/")
			proxyMu.Lock()
			u, ok := proxyStreams[id]
			proxyMu.Unlock()
			if !ok {
				googleMu.Unlock()
				http.Error(w, "not found", 404)
				return
			}

			// Parse the client's range (if any).
			var rangeStart int64 = -1
			if rng := r.Header.Get("Range"); rng != "" {
				fmt.Sscanf(rng, "bytes=%d", &rangeStart)
			}

			// Determine total size from a small probe.
			totalSize := int64(0)
			probeReq, _ := http.NewRequest("GET", u, nil)
			probeReq.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_7_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15")
			probeReq.Header.Set("Range", "bytes=0-0")
			probeClient := &http.Client{Timeout: 5 * time.Second}
			probeResp, err := probeClient.Do(probeReq)
			if err != nil {
				googleMu.Unlock()
				http.Error(w, err.Error(), 502)
				return
			}
			cr := probeResp.Header.Get("Content-Range")
			probeResp.Body.Close()
			if parts := strings.Split(cr, "/"); len(parts) == 2 {
				fmt.Sscanf(parts[1], "%d", &totalSize)
			}
			if totalSize == 0 {
				googleMu.Unlock()
				http.Error(w, "unknown size", 502)
				return
			}
			// Release the global lock before streaming so seeks are not blocked.
			googleMu.Unlock()

			// If the client sent a Range, serve from that offset.
			if rangeStart >= 0 {
				// Stream from rangeStart to end in 1 MB chunks.
				w.Header().Set("Content-Type", "audio/mp4")
				w.Header().Set("Accept-Ranges", "bytes")
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeStart, totalSize-1, totalSize))
				w.Header().Set("Content-Length", fmt.Sprintf("%d", totalSize-rangeStart))
				w.WriteHeader(206)

				const chunkSize = 5 * 1024 * 1024
				offset := rangeStart
				for offset < totalSize {
					end := offset + chunkSize - 1
					if end >= totalSize {
						end = totalSize - 1
					}
					req, _ := http.NewRequest("GET", u, nil)
					req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_7_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15")
					req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end))
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						log.Printf("youtube: proxy chunk fetch err: %v", err)
						break
					}
					n, err := io.Copy(w, resp.Body)
					resp.Body.Close()
					if err != nil || n == 0 {
						break
					}
					offset += n
					if f, ok := w.(http.Flusher); ok {
						f.Flush()
					}
				}
				return
			}

			// No Range from client: stream the full file in 1 MB chunks as 200.
			w.Header().Set("Content-Type", "audio/mp4")
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", totalSize))
			w.WriteHeader(200)

			const chunkSize = 5 * 1024 * 1024
			offset := int64(0)
			for offset < totalSize {
				end := offset + chunkSize - 1
				if end >= totalSize {
					end = totalSize - 1
				}
				req, _ := http.NewRequest("GET", u, nil)
				req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_7_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15")
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, end))
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Printf("youtube: proxy chunk fetch err: %v", err)
					break
				}
				n, err := io.Copy(w, resp.Body)
				resp.Body.Close()
				if err != nil || n == 0 {
					break
				}
				offset += n
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		})
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

// ProxyURL registers a googlevideo URL and returns the local proxy URL for it.
func ProxyURL(videoID, googlevideoURL string) string {
	StartProxy()
	proxyMu.Lock()
	proxyStreams[videoID] = googlevideoURL
	proxyMu.Unlock()
	return fmt.Sprintf("http://127.0.0.1:%d/stream/%s", proxyPort, videoID)
}
