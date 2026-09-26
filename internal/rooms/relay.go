package rooms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const maxMessageBytes = 16 << 20

// probeRoom is a code that only exists to test a token. /rooms is read-only, so
// an unknown code answers {"active":false} without creating or touching a room.
const probeRoom = "jota-probe"

// ErrTokenRequired is returned when the relay rejects the handshake with 401,
// i.e. it has AUTH_TOKEN configured and we did not present a valid one.
var ErrTokenRequired = errors.New("relay requires a token")

// Relay is the WebSocket client for a Jota relay. Message shapes are documented
// in Jota-Music/relay's README. Keep the JSON contract in sync with
// frontend/src/lib/sync/model.
type Relay struct {
	mu     sync.Mutex
	conn   *websocket.Conn
	cancel context.CancelFunc
	gen    uint64
	notify func(string, ...any)
}

func NewRelay(notify func(string, ...any)) *Relay {
	return &Relay{notify: notify}
}

func (s *Relay) Connect(rawURL string, room string, role string, token string, password string) error {
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

	s.emit("sync:connected")
	go s.read(runCtx, conn)
	return nil
}

func (s *Relay) Send(payload string) error {
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

func (s *Relay) Stop() {
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

func (s *Relay) read(ctx context.Context, conn *websocket.Conn) {
	defer func() {
		s.mu.Lock()
		active := s.conn == conn
		if active {
			s.conn = nil
			s.cancel = nil
		}
		s.mu.Unlock()
		if active {
			s.emit("sync:closed")
		}
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		s.emit("sync:message", string(data))
	}
}

// Check probes /healthz. It returns whether the relay requires an auth token
// and, when it does, whether the token we hold is the one it accepts.
// Older relays return a plain 200 body, in which case a token is assumed not to
// be required.
func (s *Relay) Check(rawURL string, token string) (bool, error) {
	target, err := healthURL(rawURL)
	if err != nil {
		return false, err
	}
	status, body, err := get(target, "")
	if err != nil {
		return false, err
	}
	if status != http.StatusOK {
		return false, statusError(status)
	}
	var health struct {
		Auth bool `json:"auth"`
	}
	if err := json.Unmarshal(body, &health); err != nil {
		return false, nil
	}
	if !health.Auth {
		return false, nil
	}
	if token == "" {
		return true, ErrTokenRequired
	}
	return true, probeToken(rawURL, token)
}

// probeToken asks /rooms for a code nobody holds, which the relay answers the
// same way it gates /ws: with 401 when the bearer is not the configured one.
// Anything else leaves the token unjudged (an unknown code answers 200, and a
// relay without the endpoint answers 404): /healthz already proved the relay
// is there, so there is nothing left to report.
func probeToken(rawURL string, token string) error {
	target, err := roomStatusURL(rawURL, probeRoom)
	if err != nil {
		return nil
	}
	status, _, err := get(target, token)
	if err != nil {
		return nil
	}
	if status == http.StatusUnauthorized {
		return ErrTokenRequired
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

// RoomStatus asks whether a room is live and how many members it has. It never
// joins: only the codes the user saved are queried, so it cannot disturb a room.
// A relay without this endpoint answers 404, which callers show as "unknown".
func (s *Relay) RoomStatus(rawURL string, room string, token string) (RoomStatus, error) {
	target, err := roomStatusURL(rawURL, room)
	if err != nil {
		return RoomStatus{}, err
	}
	status, body, err := get(target, token)
	if err != nil {
		return RoomStatus{}, err
	}
	if status != http.StatusOK {
		return RoomStatus{}, statusError(status)
	}
	var out RoomStatus
	if err := json.Unmarshal(body, &out); err != nil {
		return RoomStatus{}, err
	}
	return out, nil
}

// get reads a relay endpoint with the shared timeout and auth header, so every
// probe reaches the relay the same way. It returns the status and a bounded
// body; the caller never has to close anything.
func get(target string, token string) (int, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, body, nil
}

func statusError(status int) error {
	return fmt.Errorf("relay returned %d %s", status, http.StatusText(status))
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

func (s *Relay) emit(name string, data ...any) {
	if s.notify == nil {
		return
	}
	s.notify(name, data...)
}
