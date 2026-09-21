package covers

import (
	"bytes"
	"container/list"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/image/draw"

	_ "image/png"
)

const (
	path          = "/__img"
	minWidth      = 32
	maxWidth      = 640
	defaultWidth  = 300
	maxSource     = 8 << 20
	quality       = 82
	fetchTimeout  = 10 * time.Second
	maxConcurrent = 16
	cacheBudget   = 16 << 20
	userAgent     = "Mozilla/5.0 (compatible; Jota/1.0)"
)

// Sizes the UI actually renders. Rounding every request up to one of these
// makes nearby components share a cache entry instead of each downloading and
// resizing its own copy of the same cover.
var buckets = []int{80, 128, 160, 320, 480, 640}

// Server resizes remote cover art to the exact size the UI shows, so the
// webview never decodes a full-resolution image for a small thumbnail. Rendered
// covers are kept in a small bounded memory cache (never on disk), on top of
// whatever the webview caches for itself.
type Server struct {
	client *http.Client
	sem    chan struct{}
	cache  *lru
}

func New() *Server {
	return &Server{
		client: &http.Client{Timeout: fetchTimeout},
		sem:    make(chan struct{}, maxConcurrent),
		cache:  newLRU(cacheBudget),
	}
}

// Middleware intercepts /__img requests and forwards everything else to next.
func (s *Server) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			next.ServeHTTP(w, r)
			return
		}
		s.serve(w, r)
	})
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	target, ok := allowed(r.URL.Query().Get("u"))
	if !ok {
		http.Error(w, "invalid cover url", http.StatusBadRequest)
		return
	}
	width := bucketWidth(clampWidth(r.URL.Query().Get("w")))
	key := strconv.Itoa(width) + "|" + target

	data, ok := s.cache.get(key)
	if !ok {
		rendered, err := s.render(r.Context(), target, width)
		if err != nil {
			// The proxy is best-effort: fall back to the original URL so art
			// still shows even when the upstream fetch or decode fails.
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
		data = rendered
		s.cache.put(key, data)
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(data)
}

func (s *Server) render(ctx context.Context, target string, width int) ([]byte, error) {
	s.sem <- struct{}{}
	img, err := s.fetch(ctx, target)
	if err != nil {
		<-s.sem
		return nil, err
	}
	resized := resize(img, width)
	<-s.sem

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *Server) fetch(ctx context.Context, target string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	// Ask for formats the stdlib can decode; without this, Google's CDN may
	// answer with WebP, which image.Decode does not understand.
	req.Header.Set("Accept", "image/jpeg,image/png,image/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cover: %s returned %d", target, resp.StatusCode)
	}

	img, _, err := image.Decode(io.LimitReader(resp.Body, maxSource))
	return img, err
}

func resize(img image.Image, width int) image.Image {
	bounds := img.Bounds()
	if bounds.Dx() <= width {
		return img
	}
	height := max(width*bounds.Dy()/bounds.Dx(), 1)
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

func clampWidth(raw string) int {
	width, err := strconv.Atoi(raw)
	if err != nil || width <= 0 {
		return defaultWidth
	}
	if width < minWidth {
		return minWidth
	}
	if width > maxWidth {
		return maxWidth
	}
	return width
}

func bucketWidth(width int) int {
	for _, bucket := range buckets {
		if width <= bucket {
			return bucket
		}
	}
	return maxWidth
}

// lru is a byte-budgeted least-recently-used cache of rendered covers. It is
// deliberately tiny and internal: one lock, a map and an eviction list.
type lru struct {
	mu    sync.Mutex
	items map[string]*list.Element
	order *list.List
	bytes int
	max   int
}

type lruItem struct {
	key  string
	data []byte
}

func newLRU(max int) *lru {
	return &lru{
		items: make(map[string]*list.Element),
		order: list.New(),
		max:   max,
	}
}

func (c *lru) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(elem)
	return elem.Value.(*lruItem).data, true
}

func (c *lru) put(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		item := elem.Value.(*lruItem)
		c.bytes += len(data) - len(item.data)
		item.data = data
		c.order.MoveToFront(elem)
	} else {
		item := &lruItem{key: key, data: data}
		c.items[key] = c.order.PushFront(item)
		c.bytes += len(data)
	}

	for c.bytes > c.max {
		elem := c.order.Back()
		if elem == nil {
			break
		}
		item := elem.Value.(*lruItem)
		c.order.Remove(elem)
		delete(c.items, item.key)
		c.bytes -= len(item.data)
	}
}
