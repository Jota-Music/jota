package youtube

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
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
		MaxIdleConnsPerHost:   8,
	},
}

const proxyAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_7_3) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.0 Safari/605.1.15"

// chunkSize is the largest slice requested from googlevideo in one go. YouTube
// throttles a single long connection to a few tens of KB/s, so a track is
// fetched as a sequence of short requests that each get a full-speed burst.
const chunkSize = 5 * 1024 * 1024

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

// relay answers one player request. The byte range googlevideo serves is
// reassembled from short, bounded requests so the throttled long connection is
// avoided, while the status, offsets and headers describe exactly what the
// player asked for.
func relay(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/stream/")
	proxyMu.Lock()
	s, ok := proxyStreams[id]
	proxyMu.Unlock()
	if !ok {
		http.Error(w, "not found", 404)
		return
	}

	// Ask googlevideo for a single byte first: its Content-Range carries the
	// real total size, which is what the headers and the chunking need.
	probeReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, s.url, nil)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	probeReq.Header.Set("User-Agent", proxyAgent)
	probeReq.Header.Set("Range", "bytes=0-0")
	probeClient := &http.Client{Timeout: 10 * time.Second, Transport: proxyClient.Transport}
	probeResp, err := probeClient.Do(probeReq)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		log.Printf("youtube: proxy %s probe err=%v", id, err)
		http.Error(w, "upstream unavailable", 502)
		return
	}

	total := int64(0)
	if parts := strings.Split(probeResp.Header.Get("Content-Range"), "/"); len(parts) == 2 {
		fmt.Sscanf(parts[1], "%d", &total)
	}
	if total == 0 {
		if cl := probeResp.Header.Get("Content-Length"); cl != "" {
			fmt.Sscanf(cl, "%d", &total)
		}
	}
	mime := s.mime
	if ct := probeResp.Header.Get("Content-Type"); ct != "" {
		mime = ct
	}
	_, _ = io.Copy(io.Discard, probeResp.Body)
	probeResp.Body.Close()

	if total <= 0 {
		// Fallback: pass the request through untouched when the size cannot be
		// learned. Slower on throttled streams, but keeps playback working.
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
		return
	}

	// The range the player asked for. An open-ended request runs to the end.
	start := int64(0)
	end := total - 1
	hasRange := false
	if rng := r.Header.Get("Range"); rng != "" {
		rs, re, ok := parseRange(rng)
		if ok {
			hasRange = true
			start = rs
			if re >= 0 {
				end = re
			}
			if end >= total {
				end = total - 1
			}
			if start > end {
				w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", total))
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
		}
	}

	if hasRange {
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, total))
		w.Header().Set("Content-Length", strconv.FormatInt(end-start+1, 10))
		w.WriteHeader(http.StatusPartialContent)
	} else {
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.FormatInt(total, 10))
		w.WriteHeader(http.StatusOK)
	}

	flusher, _ := w.(http.Flusher)
	chunkClient := &http.Client{Timeout: 30 * time.Second, Transport: proxyClient.Transport}

	offset := start
	delivered := int64(0)
	expected := end - start + 1
	for offset <= end {
		ce := offset + chunkSize - 1
		if ce > end {
			ce = end
		}
		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, s.url, nil)
		if err != nil {
			log.Printf("youtube: proxy %s err=%v", id, err)
			break
		}
		req.Header.Set("User-Agent", proxyAgent)
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, ce))
		resp, err := chunkClient.Do(req)
		if err != nil {
			if r.Context().Err() != nil {
				return
			}
			log.Printf("youtube: proxy %s err=%v", id, err)
			break
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			log.Printf("youtube: proxy %s upstream status %d", id, resp.StatusCode)
			resp.Body.Close()
			break
		}
		n, err := io.Copy(w, resp.Body)
		resp.Body.Close()
		if err != nil {
			if r.Context().Err() == nil {
				log.Printf("youtube: proxy %s copy err=%v", id, err)
			}
			break
		}
		if n == 0 {
			break
		}
		offset += n
		delivered += n
		if flusher != nil {
			flusher.Flush()
		}
	}
	if delivered < expected {
		log.Printf("youtube: proxy %s short response: got %d of %d bytes", id, delivered, expected)
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

func parseRange(s string) (int64, int64, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "bytes=") {
		return 0, 0, false
	}
	spec := strings.TrimPrefix(s, "bytes=")
	if i := strings.IndexByte(spec, ','); i >= 0 {
		spec = spec[:i]
	}
	parts := strings.SplitN(spec, "-", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	var start int64
	if parts[0] != "" {
		v, err := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil || v < 0 {
			return 0, 0, false
		}
		start = v
	} else {
		// Suffix ranges are not supported here.
		return 0, 0, false
	}
	end := int64(-1)
	if parts[1] != "" {
		v, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err != nil || v < start {
			return 0, 0, false
		}
		end = v
	}
	return start, end, true
}
