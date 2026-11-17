package youtube

import (
	"context"
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
	// at is when the entry was registered, so the registry can drop the oldest
	// one instead of growing with every video played.
	at time.Time
}

// proxyStreamLimit bounds the registry. Each entry is a signed URL of about a
// kilobyte, and a session plays far more videos than it ever serves at once.
const proxyStreamLimit = 256

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

// chunkTimeout is how long one chunk may take. It is generous on purpose: it
// only cuts a link that stalls, and a chunk is five megabytes, so a strict
// deadline here would cut the playback of anyone on a slow connection.
const chunkTimeout = 3 * time.Minute

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

// probeTimeout bounds the answer to a header-only request: one byte is all it
// asks for, so a probe that outlives this is a link that will not play.
const probeTimeout = 10 * time.Second

// relay answers one player request. The slice it serves is also the request
// that learns the stream's own size: googlevideo's Content-Range carries the
// whole length, so a play costs one round trip instead of a probe plus the
// slice. On a phone that round trip is most of what a track's start, and every
// seek after it, is made of.
func relay(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/stream/")
	proxyMu.Lock()
	s, ok := proxyStreams[id]
	proxyMu.Unlock()
	if !ok {
		http.Error(w, "not found", 404)
		return
	}

	if r.Method == http.MethodHead {
		head(w, r, s)
		return
	}

	// The range the player asked for. An open-ended request runs to the end of
	// the stream, which is only known once googlevideo answers.
	start := int64(0)
	end := int64(-1)
	hasRange := false
	if rs, re, ok := parseRange(r.Header.Get("Range")); ok {
		hasRange, start, end = true, rs, re
	}

	// The first slice is the probe: it is asked for before the headers because
	// its answer is what tells them how long the stream is.
	ce := start + chunkSize - 1
	if end >= 0 && end < ce {
		ce = end
	}
	ctx, cancel := context.WithTimeout(r.Context(), chunkTimeout)
	defer cancel()
	resp, err := slice(ctx, s.url, start, ce)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		log.Printf("youtube: proxy %s err=%v", id, err)
		http.Error(w, "upstream unavailable", 502)
		return
	}

	total := size(resp)
	mime := s.mime
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		mime = ct
	}

	switch {
	case resp.StatusCode == http.StatusRequestedRangeNotSatisfiable:
		// A seek past the end: the answer already says how long the stream is.
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			w.Header().Set("Content-Range", cr)
		}
		resp.Body.Close()
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	case resp.StatusCode == http.StatusOK && start > 0:
		// The whole object under a range from the middle: its bytes do not
		// start where the headers would say, so there is nothing to serve.
		log.Printf("youtube: proxy %s upstream ignored the range", id)
		resp.Body.Close()
		http.Error(w, "upstream unavailable", 502)
		return
	case resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent:
		log.Printf("youtube: proxy %s upstream status %d", id, resp.StatusCode)
		resp.Body.Close()
		http.Error(w, "upstream unavailable", 502)
		return
	}

	if total <= 0 {
		// Without the size the slicing cannot describe what it sends: pass this
		// same answer through instead of asking for it twice.
		pass(w, resp, s.mime)
		return
	}

	// The end the player asked for, or the end of the stream.
	if end < 0 || end > total-1 {
		end = total - 1
	}
	if start > end {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", total))
		resp.Body.Close()
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
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
	expected := end - start + 1

	// The first slice is already open, so it is copied before the next one is
	// asked for: a track that fits in one chunk costs a single request.
	n, copyErr := io.Copy(w, io.LimitReader(resp.Body, expected))
	resp.Body.Close()
	if copyErr != nil && r.Context().Err() == nil {
		log.Printf("youtube: proxy %s copy err=%v", id, copyErr)
	}
	delivered := n
	if flusher != nil {
		flusher.Flush()
	}

	offset := start + n
	for copyErr == nil && n > 0 && delivered < expected {
		ce := offset + chunkSize - 1
		if ce > end {
			ce = end
		}
		// The deadline is per chunk, not for the whole body: a chunk is five
		// megabytes, and a link slow enough to take longer than this is the only
		// one that should be cut, not every link under a few megabits.
		ctx, cancel := context.WithTimeout(r.Context(), chunkTimeout)
		res, err := slice(ctx, s.url, offset, ce)
		if err != nil {
			cancel()
			if r.Context().Err() != nil {
				return
			}
			log.Printf("youtube: proxy %s err=%v", id, err)
			break
		}
		// A 200 to a ranged request is the whole object, not this chunk: copying
		// it from a middle offset would write the wrong bytes under a length that
		// says otherwise.
		if res.StatusCode == http.StatusOK && offset > start {
			log.Printf("youtube: proxy %s upstream ignored the range", id)
			res.Body.Close()
			cancel()
			break
		}
		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusPartialContent {
			log.Printf("youtube: proxy %s upstream status %d", id, res.StatusCode)
			res.Body.Close()
			cancel()
			break
		}
		// Only what the caller asked for: an answer that carries more than the
		// chunk must not overflow the length already written in the header.
		room := end - offset + 1
		n, err = io.Copy(w, io.LimitReader(res.Body, room))
		res.Body.Close()
		cancel()
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

// head answers a header-only player request. A one-byte range is the probe
// googlevideo always answers, and nothing more of the stream is pulled.
func head(w http.ResponseWriter, r *http.Request, s stream) {
	ctx, cancel := context.WithTimeout(r.Context(), probeTimeout)
	defer cancel()
	resp, err := slice(ctx, s.url, 0, 0)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		http.Error(w, "upstream unavailable", 502)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		http.Error(w, "upstream unavailable", 502)
		return
	}

	mime := s.mime
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		mime = ct
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Accept-Ranges", "bytes")
	if total := size(resp); total > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(total, 10))
	}
	w.WriteHeader(http.StatusOK)
}

// pass hands an upstream answer through untouched. It is the last resort for a
// stream whose size cannot be learned, which is when the range slicing cannot
// describe what it sends.
func pass(w http.ResponseWriter, resp *http.Response, fallback string) {
	defer resp.Body.Close()
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Last-Modified", "ETag"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", fallback)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("youtube: proxy copy err=%v", err)
	}
}

// slice asks googlevideo for one bounded slice of a stream, with the headers
// the player will be answered from.
func slice(ctx context.Context, raw string, start, end int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", proxyAgent)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	return proxyClient.Do(req)
}

// size is the whole stream's length, as the answer to a slice advertises it: a
// 206 carries it in Content-Range, and a 200 in Content-Length.
func size(resp *http.Response) int64 {
	if parts := strings.Split(resp.Header.Get("Content-Range"), "/"); len(parts) == 2 {
		if total, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64); err == nil {
			return total
		}
	}
	if resp.StatusCode == http.StatusOK {
		if total, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64); err == nil {
			return total
		}
	}
	return 0
}

// ProxyURL registers a googlevideo URL and returns the local proxy URL for it.
func ProxyURL(videoID, googlevideoURL string) string {
	port := StartProxy()
	if port == 0 {
		// No relay to hand the player: the URL the source signed still plays, it
		// just does not get the range slicing. A silent http://127.0.0.1:0 would
		// fail the playback with no reason in the log.
		return googlevideoURL
	}
	proxyMu.Lock()
	proxyStreams[videoID] = stream{
		url:  unpinned(googlevideoURL),
		mime: streamMime(googlevideoURL),
		at:   time.Now(),
	}
	if len(proxyStreams) > proxyStreamLimit {
		var oldest string
		var at time.Time
		for id, s := range proxyStreams {
			if oldest == "" || s.at.Before(at) {
				oldest, at = id, s.at
			}
		}
		delete(proxyStreams, oldest)
	}
	proxyMu.Unlock()
	return fmt.Sprintf("http://127.0.0.1:%d/stream/%s", port, videoID)
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
