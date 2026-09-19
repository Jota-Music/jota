package discord

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
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
)

type Service struct {
	clientID string
	mu       sync.Mutex
	writeMu  sync.Mutex
	conn     net.Conn
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
	s.mu.Lock()
	defer s.mu.Unlock()
	conn := s.conn
	s.conn = nil
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (s *Service) connect() (net.Conn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		return s.conn, nil
	}
	conn, err := platformDial()
	if err != nil {
		return nil, err
	}
	s.conn = conn
	handshake, err := json.Marshal(map[string]any{"v": 1, "client_id": s.clientID})
	if err != nil {
		s.invalidateLocked(conn)
		return nil, err
	}
	if err := s.frame(conn, opHandshake, handshake); err != nil {
		s.invalidateLocked(conn)
		return nil, err
	}
	if err := readHandshake(conn); err != nil {
		s.invalidateLocked(conn)
		return nil, err
	}
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
	payload := make([]byte, binary.LittleEndian.Uint32(header[4:8]))
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
