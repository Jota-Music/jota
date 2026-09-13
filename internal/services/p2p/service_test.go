package p2p

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
)

func waitOpen(t *testing.T, s *Service) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		dc := s.dc
		s.mu.Unlock()
		if dc != nil && dc.ReadyState() == webrtc.DataChannelStateOpen {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("data channel did not open")
}

func connect(t *testing.T) (*Service, *Service) {
	t.Helper()
	host := New(Config{})
	guest := New(Config{})

	hostCode, err := host.StartHost()
	if err != nil {
		t.Fatalf("start host: %v", err)
	}
	guestCode, err := guest.JoinGuest(hostCode)
	if err != nil {
		t.Fatalf("join guest: %v", err)
	}
	if err := host.AcceptAnswer(guestCode); err != nil {
		t.Fatalf("accept answer: %v", err)
	}

	waitOpen(t, host)
	waitOpen(t, guest)
	return host, guest
}

func TestPeerDataChannel(t *testing.T) {
	host, guest := connect(t)
	defer host.Stop()
	defer guest.Stop()

	var mu sync.Mutex
	hostGot := []string{}
	guestGot := []string{}

	host.mu.Lock()
	hostDC := host.dc
	host.mu.Unlock()

	guest.mu.Lock()
	guestDC := guest.dc
	guest.mu.Unlock()

	hostDC.OnMessage(func(m webrtc.DataChannelMessage) {
		mu.Lock()
		hostGot = append(hostGot, string(m.Data))
		mu.Unlock()
	})
	guestDC.OnMessage(func(m webrtc.DataChannelMessage) {
		mu.Lock()
		guestGot = append(guestGot, string(m.Data))
		mu.Unlock()
	})

	if err := host.Send("from-host"); err != nil {
		t.Fatalf("host send: %v", err)
	}
	if err := guest.Send("from-guest"); err != nil {
		t.Fatalf("guest send: %v", err)
	}

	time.Sleep(2 * time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(guestGot) == 0 {
		t.Fatalf("guest received nothing; hostGot=%v", hostGot)
	}
	if len(hostGot) == 0 {
		t.Fatalf("host received nothing; guestGot=%v", guestGot)
	}
	t.Logf("guestGot=%v hostGot=%v", guestGot, hostGot)
}

func TestPeerChunkSize(t *testing.T) {
	host, guest := connect(t)
	defer host.Stop()
	defer guest.Stop()

	payload := strings.Repeat("x", 64*1024)

	guest.mu.Lock()
	dc := guest.dc
	guest.mu.Unlock()

	var mu sync.Mutex
	received := -1
	dc.OnMessage(func(m webrtc.DataChannelMessage) {
		mu.Lock()
		received = len(m.Data)
		mu.Unlock()
	})

	if err := host.Send(payload); err != nil {
		t.Fatalf("send chunk: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := received
		mu.Unlock()
		if n >= 0 {
			if n != len(payload) {
				t.Fatalf("received %d bytes, want %d", n, len(payload))
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("chunk never received")
}

func TestPeerManyChunks(t *testing.T) {
	host, guest := connect(t)
	defer host.Stop()
	defer guest.Stop()

	guest.mu.Lock()
	dc := guest.dc
	guest.mu.Unlock()

	var mu sync.Mutex
	count := 0

	dc.OnMessage(func(m webrtc.DataChannelMessage) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	const chunks = 40
	const size = 64 * 1024
	for i := 0; i < chunks; i++ {
		if err := host.Send(strings.Repeat("x", size)); err != nil {
			t.Fatalf("send chunk %d: %v", i, err)
		}
	}

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		c := count
		mu.Unlock()
		if c >= chunks {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	mu.Lock()
	c := count
	mu.Unlock()
	t.Fatalf("received %d of %d chunks", c, chunks)
}
