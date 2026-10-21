package spotify

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/devgianlu/go-librespot/ap"
)

// A stalled accesspoint is what used to make the app open without a window: the
// connection only carries a deadline when the caller's context has one, so a
// restore with an unbounded context parks inside receivePacket forever.
// SessionTimeout is only worth having if librespot honours the deadline; this
// fails the day it stops, which is when the bound has to move into our own code.
func TestStalledAccesspointHonoursDeadline(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Hold every accepted connection open and never reply: a blackholed peer.
	var mu sync.Mutex
	var held []net.Conn
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			held = append(held, conn)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range held {
			_ = conn.Close()
		}
	})

	stalled := ap.NewAccesspoint(
		&browserLogger{},
		func(context.Context) string { return ln.Addr().String() },
		"0123456789abcdef0123456789abcdef01234567",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- stalled.ConnectStored(ctx, "listener", []byte("stale-credentials"))
	}()

	start := time.Now()
	select {
	case connectErr := <-done:
		elapsed := time.Since(start)
		if connectErr == nil {
			t.Fatal("a stalled accesspoint reported a successful login")
		}
		if elapsed > SessionTimeout {
			t.Fatalf("connect outlived its context: %s (err %v)", elapsed, connectErr)
		}
		if elapsed < time.Second {
			t.Fatalf("connect gave up before the deadline, so it proves nothing: %s", elapsed)
		}
	case <-time.After(SessionTimeout + 5*time.Second):
		t.Fatal("accesspoint ignored the context deadline; the restore would hang forever")
	}
}

func TestUsernameSurvivesDegraded(t *testing.T) {
	s := NewSpotifyService("")
	if got := s.Username(); got != "" {
		t.Fatalf("fresh service username = %q, want empty", got)
	}

	s.username = "listener"
	s.degraded = true

	if got := s.Username(); got != "listener" {
		t.Errorf("degraded username = %q, want %q", got, "listener")
	}
	if s.IsConnected() {
		t.Error("degraded service reports connected; writes need a live session")
	}
	if !s.IsDegraded() {
		t.Error("IsDegraded() = false, want true")
	}
}

func TestDegradedRequiresStoredAccount(t *testing.T) {
	s := NewSpotifyService("")
	if s.IsDegraded() {
		t.Error("a service with no stored credentials must not claim to be degraded")
	}
	if s.isFinished() {
		t.Error("fresh service reports finished")
	}
}
