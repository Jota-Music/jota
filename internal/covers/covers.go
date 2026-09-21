package covers

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strconv"
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
	maxConcurrent = 4
	userAgent     = "Mozilla/5.0 (compatible; Jota/1.0)"
)

// Server resizes remote cover art to the exact size the UI shows, so the
// webview never decodes a full-resolution image for a small thumbnail. It
// keeps no state: WebKit caches the response like any other asset.
type Server struct {
	client *http.Client
	sem    chan struct{}
}

func New() *Server {
	return &Server{
		client: &http.Client{Timeout: fetchTimeout},
		sem:    make(chan struct{}, maxConcurrent),
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
	width := clampWidth(r.URL.Query().Get("w"))

	s.sem <- struct{}{}
	defer func() { <-s.sem }()

	img, err := s.fetch(r.Context(), target)
	if err != nil {
		// The proxy is best-effort: fall back to the original URL so art still
		// shows even when the upstream fetch or decode fails.
		http.Redirect(w, r, target, http.StatusFound)
		return
	}

	resized := resize(img, width)

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := jpeg.Encode(w, resized, &jpeg.Options{Quality: quality}); err != nil {
		// Response is already streaming; there is nothing left to recover.
		_ = err
	}
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
