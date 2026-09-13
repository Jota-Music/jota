package sync

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const maxMessageBytes = 16 << 20

// Message shapes are documented in Jota-Music/relay's README. Keep the JSON
// contract in sync with frontend/src/lib/sync/model.
type Service struct {
	mu     sync.Mutex
	conn   *websocket.Conn
	cancel context.CancelFunc
}

func New() *Service {
	return &Service{}
}

func (s *Service) Connect(rawURL string, room string, role string) error {
	target, err := endpoint(rawURL, room, role)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, _, err := websocket.Dial(ctx, target, nil)
	cancel()
	if err != nil {
		return err
	}
	conn.SetReadLimit(maxMessageBytes)

	s.mu.Lock()
	if s.conn != nil {
		s.mu.Unlock()
		conn.Close(websocket.StatusNormalClosure, "")
		return errors.New("already connected")
	}
	runCtx, runCancel := context.WithCancel(context.Background())
	s.conn = conn
	s.cancel = runCancel
	s.mu.Unlock()

	emit("sync:connected")
	go s.read(runCtx, conn)
	return nil
}

func (s *Service) Send(payload string) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()

	if conn == nil {
		return errors.New("not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, []byte(payload))
}

func (s *Service) Stop() {
	s.mu.Lock()
	conn := s.conn
	cancel := s.cancel
	s.conn = nil
	s.cancel = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if conn != nil {
		conn.Close(websocket.StatusNormalClosure, "")
	}
}

func (s *Service) read(ctx context.Context, conn *websocket.Conn) {
	defer func() {
		s.mu.Lock()
		active := s.conn == conn
		if active {
			s.conn = nil
			s.cancel = nil
		}
		s.mu.Unlock()
		if active {
			emit("sync:closed")
		}
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		emit("sync:message", string(data))
	}
}

func (s *Service) Check(rawURL string) error {
	target, err := healthURL(rawURL)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("relay returned %s", resp.Status)
	}
	return nil
}

func endpoint(raw string, room string, role string) (string, error) {
	if room == "" {
		return "", errors.New("room code is empty")
	}
	u, err := relayURL(raw)
	if err != nil {
		return "", err
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/ws"
	}

	q := u.Query()
	q.Set("room", room)
	q.Set("role", role)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func healthURL(raw string) (string, error) {
	u, err := relayURL(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme == "wss" {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}
	u.Path = "/healthz"
	u.RawQuery = ""
	return u.String(), nil
}

func relayURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("relay url is empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "wss://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return nil, errors.New("unsupported relay url scheme")
	}
	return u, nil
}

func emit(name string, data ...any) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.Emit(name, data...)
}
