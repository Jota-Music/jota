package sync

import (
	"context"
	"encoding/json"
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

// ErrTokenRequired is returned when the relay rejects the handshake with 401,
// i.e. it has AUTH_TOKEN configured and we did not present a valid one.
var ErrTokenRequired = errors.New("relay requires a token")

// Message shapes are documented in Jota-Music/relay's README. Keep the JSON
// contract in sync with frontend/src/lib/sync/model.
type Service struct {
	mu     sync.Mutex
	conn   *websocket.Conn
	cancel context.CancelFunc
	gen    uint64
}

func New() *Service {
	return &Service{}
}

func (s *Service) Connect(rawURL string, room string, role string, token string, password string) error {
	target, err := endpoint(rawURL, room, role)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.gen++
	gen := s.gen
	old := s.conn
	oldCancel := s.cancel
	s.conn = nil
	s.cancel = nil
	s.mu.Unlock()
	if oldCancel != nil {
		oldCancel()
	}
	if old != nil {
		old.Close(websocket.StatusNormalClosure, "")
	}

	header := http.Header{}
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	if password != "" {
		header.Set("X-Room-Password", password)
	}
	var opts *websocket.DialOptions
	if len(header) > 0 {
		opts = &websocket.DialOptions{HTTPHeader: header}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, resp, err := websocket.Dial(ctx, target, opts)
	cancel()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return ErrTokenRequired
		}
		return err
	}
	conn.SetReadLimit(maxMessageBytes)

	s.mu.Lock()
	// Stop() or a newer Connect() superseded this dial: drop it silently.
	if gen != s.gen {
		s.mu.Unlock()
		conn.Close(websocket.StatusNormalClosure, "")
		return nil
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
	s.gen++
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

// Check probes /healthz. It returns whether the relay requires an auth token
// so the UI can ask for one up front. Older relays return a plain 200 body, in
// which case a token is assumed not to be required.
func (s *Service) Check(rawURL string) (bool, error) {
	target, err := healthURL(rawURL)
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("relay returned %s", resp.Status)
	}
	var body struct {
		Auth bool `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, nil
	}
	return body.Auth, nil
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

func roomStatusURL(raw string, room string) (string, error) {
	room = strings.TrimSpace(room)
	if room == "" {
		return "", errors.New("room code is empty")
	}
	u, err := relayURL(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme == "wss" {
		u.Scheme = "https"
	} else {
		u.Scheme = "http"
	}
	u.Path = "/rooms/" + url.PathEscape(room)
	u.RawQuery = ""
	return u.String(), nil
}

// RoomStatus is a saved room's live state as reported by the relay.
type RoomStatus struct {
	Active  bool `json:"active"`
	Members int  `json:"members"`
	HasHost bool `json:"hasHost"`
	Locked  bool `json:"locked"`
}

// RoomStatus asks whether a room is live and how many members it has. It never
// joins: only the codes the user saved are queried, so it cannot disturb a room.
// A relay without this endpoint answers 404, which callers show as "unknown".
func (s *Service) RoomStatus(rawURL string, room string, token string) (RoomStatus, error) {
	target, err := roomStatusURL(rawURL, room)
	if err != nil {
		return RoomStatus{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return RoomStatus{}, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return RoomStatus{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return RoomStatus{}, fmt.Errorf("relay returned %s", resp.Status)
	}
	var out RoomStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return RoomStatus{}, err
	}
	return out, nil
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
