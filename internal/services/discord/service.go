package discord

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

const (
	opHandshake = 0
	opFrame     = 1
	opPing      = 3
	opPong      = 4
	timeout     = 5 * time.Second
	// maxFrame bounds a payload length read off the socket. Discord's IPC
	// messages are tiny, so anything past this is a broken local endpoint and
	// not something worth allocating for.
	maxFrame = 1 << 20
)

type Service struct {
	clientID string
	mu       sync.Mutex
	// dialMu serializes a connect attempt: the dial happens outside mu so a hung
	// endpoint cannot hold a status read, and this keeps a burst of updates from
	// opening two connections.
	dialMu  sync.Mutex
	writeMu sync.Mutex
	conn    net.Conn
}

func New(clientID string) *Service {
	return &Service{clientID: clientID}
}

func (s *Service) Set(activity Activity) error {
	conn, err := s.connect()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"cmd":   "SET_ACTIVITY",
		"args":  map[string]any{"pid": os.Getpid(), "activity": activity},
		"nonce": nonce(),
	})
	if err != nil {
		return err
	}
	if err := s.frame(conn, opFrame, payload); err != nil {
		s.invalidate(conn)
		return err
	}
	return nil
}

func (s *Service) Clear() error {
	conn := s.current()
	if conn == nil {
		return nil
	}
	payload, err := json.Marshal(map[string]any{
		"cmd":   "SET_ACTIVITY",
		"args":  map[string]any{"pid": os.Getpid(), "activity": nil},
		"nonce": nonce(),
	})
	if err != nil {
		return err
	}
	if err := s.frame(conn, opFrame, payload); err != nil {
		s.invalidate(conn)
		return err
	}
	return nil
}

func (s *Service) Close() error {
	// The lock is held for a whole connect, including a handshake that waits
	// up to timeout on a Discord socket that may not be there. Waiting here
	// freezes the shutdown, and the window never gets destroyed: on Linux that
	// is the "force quit" prompt. Closing is best-effort, so a connect in
	// flight just leaves the socket to the process exit.
	if !s.mu.TryLock() {
		return nil
	}
	defer s.mu.Unlock()
	conn := s.conn
	s.conn = nil
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (s *Service) connect() (net.Conn, error) {
	s.dialMu.Lock()
	defer s.dialMu.Unlock()
	if conn := s.current(); conn != nil {
		return conn, nil
	}
	conn, err := platformDial()
	if err != nil {
		return nil, err
	}
	handshake, err := json.Marshal(map[string]any{"v": 1, "client_id": s.clientID})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := s.frame(conn, opHandshake, handshake); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := readHandshake(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	// Published once it answered the handshake, so no caller is handed a
	// connection that is not usable yet.
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	go s.listen(conn)
	return conn, nil
}

func (s *Service) listen(conn net.Conn) {
	for {
		op, payload, err := readFrame(conn)
		if err != nil {
			s.invalidate(conn)
			return
		}
		if op == opPing {
			if err := s.frame(conn, opPong, payload); err != nil {
				s.invalidate(conn)
				return
			}
		}
	}
}

func (s *Service) frame(conn net.Conn, op uint32, payload []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	msg := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(msg[0:4], op)
	binary.LittleEndian.PutUint32(msg[4:8], uint32(len(payload)))
	copy(msg[8:], payload)
	_, err := conn.Write(msg)
	return err
}

func (s *Service) current() net.Conn {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn
}

func (s *Service) invalidate(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalidateLocked(conn)
}

func (s *Service) invalidateLocked(conn net.Conn) {
	if s.conn == conn {
		s.conn = nil
	}
	_ = conn.Close()
}

func readHandshake(conn net.Conn) error {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	_, _, err := readFrame(conn)
	_ = conn.SetReadDeadline(time.Time{})
	return err
}

func readFrame(conn net.Conn) (uint32, []byte, error) {
	var header [8]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return 0, nil, err
	}
	op := binary.LittleEndian.Uint32(header[0:4])
	length := binary.LittleEndian.Uint32(header[4:8])
	if length > maxFrame {
		return 0, nil, errors.New("discord: frame too large")
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return 0, nil, err
	}
	return op, payload, nil
}

func nonce() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b[:])
}
