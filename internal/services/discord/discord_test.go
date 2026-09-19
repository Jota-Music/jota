package discord

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"testing"
)

func readFrameJSON(conn net.Conn) (map[string]any, error) {
	var header [8]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint32(header[0:4]) != opFrame {
		return nil, nil
	}
	body := make([]byte, binary.LittleEndian.Uint32(header[4:8]))
	if _, err := io.ReadFull(conn, body); err != nil {
		return nil, err
	}
	msg := map[string]any{}
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func TestSetFrames(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	type result struct {
		msg map[string]any
		err error
	}
	ch := make(chan result, 1)
	go func() {
		msg, err := readFrameJSON(server)
		ch <- result{msg, err}
	}()

	s := &Service{conn: client}
	if err := s.Set(Activity{Type: 2, Details: "Song", State: "Artist"}); err != nil {
		t.Fatal(err)
	}

	r := <-ch
	if r.err != nil {
		t.Fatal(r.err)
	}
	if r.msg["cmd"] != "SET_ACTIVITY" {
		t.Fatalf("cmd = %v, want SET_ACTIVITY", r.msg["cmd"])
	}
	if r.msg["nonce"] == "" {
		t.Fatal("missing nonce")
	}
	args := r.msg["args"].(map[string]any)
	activity := args["activity"].(map[string]any)
	if activity["type"].(float64) != 2 {
		t.Fatalf("type = %v, want 2", activity["type"])
	}
	if activity["details"] != "Song" || activity["state"] != "Artist" {
		t.Fatalf("unexpected activity: %v", activity)
	}
}

func TestClearFrames(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	type result struct {
		msg map[string]any
		err error
	}
	ch := make(chan result, 1)
	go func() {
		msg, err := readFrameJSON(server)
		ch <- result{msg, err}
	}()

	s := &Service{conn: client}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}

	r := <-ch
	if r.err != nil {
		t.Fatal(r.err)
	}
	args := r.msg["args"].(map[string]any)
	if args["activity"] != nil {
		t.Fatalf("activity = %v, want null", args["activity"])
	}
}

func encodeFrame(op uint32, payload []byte) []byte {
	msg := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(msg[0:4], op)
	binary.LittleEndian.PutUint32(msg[4:8], uint32(len(payload)))
	copy(msg[8:], payload)
	return msg
}

func TestListenPongsPing(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	s := &Service{conn: client}
	go s.listen(client)

	payload := []byte(`{"ping":1}`)
	if _, err := server.Write(encodeFrame(opPing, payload)); err != nil {
		t.Fatal(err)
	}

	op, body, err := readFrame(server)
	if err != nil {
		t.Fatal(err)
	}
	if op != opPong {
		t.Fatalf("opcode = %d, want %d", op, opPong)
	}
	if string(body) != string(payload) {
		t.Fatalf("payload = %s, want %s", body, payload)
	}
}
