package single

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestSecondAcquireHandsOff(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())

	focused := make(chan struct{}, 1)
	first, err := Acquire(func() { focused <- struct{}{} })
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer first.Close()

	if _, err := Acquire(func() {}); !errors.Is(err, ErrHandedOff) {
		t.Fatalf("second acquire = %v, want ErrHandedOff", err)
	}

	select {
	case <-focused:
	case <-time.After(2 * time.Second):
		t.Fatal("the running instance was never asked to show its window")
	}
}

// The instance this package exists for: alive, holding the socket, never
// answering. Waiting on it is the hang; it has to be replaced instead, or it
// keeps the store lock and every launch stays mute.
func TestAcquireReplacesUnresponsiveInstance(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	wedged, err := net.Listen("unix", filepath.Join(dir, "jota.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer wedged.Close()

	child := exec.Command(sleeper(t, dir), "60")
	if err := child.Start(); err != nil {
		t.Skipf("cannot start a stand-in process: %v", err)
	}
	defer func() { _ = child.Process.Kill() }()

	pid := strconv.Itoa(child.Process.Pid)
	if err := os.WriteFile(filepath.Join(dir, "jota.sock.pid"), []byte(pid), 0o644); err != nil {
		t.Fatal(err)
	}

	guard, err := Acquire(func() {})
	if err != nil {
		t.Fatalf("acquire over an unresponsive instance: %v", err)
	}
	defer guard.Close()

	waited := make(chan error, 1)
	go func() { waited <- child.Wait() }()
	select {
	case <-waited:
	case <-time.After(5 * time.Second):
		t.Fatal("the unresponsive instance was never asked to die")
	}
}

// A pid file outlives its owner, and the pid it names can be reused: the guard
// must not signal the stranger that inherited it.
func TestAcquireLeavesStrangerAlive(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	child := exec.Command("sleep", "60")
	if err := child.Start(); err != nil {
		t.Skipf("cannot start a stand-in process: %v", err)
	}
	defer func() { _ = child.Process.Kill() }()

	if err := os.WriteFile(filepath.Join(dir, "jota.sock.pid"), []byte(strconv.Itoa(child.Process.Pid)), 0o644); err != nil {
		t.Fatal(err)
	}

	guard, err := Acquire(func() {})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer guard.Close()

	if err := child.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf("a stranger holding a reused pid was signalled: %v", err)
	}
}

// sleeper copies sleep under the name jota: the guard only kills a pid that still
// runs jota, so a stand-in called sleep is rightly left alone.
func sleeper(t *testing.T, dir string) string {
	t.Helper()
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("no sleep binary to stand in with: %v", err)
	}
	data, err := os.ReadFile(sleep)
	if err != nil {
		t.Skipf("cannot read %s: %v", sleep, err)
	}
	path := filepath.Join(dir, "jota")
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Skipf("cannot write the stand-in: %v", err)
	}
	return path
}

// A second launch arriving while this instance closes must not be handed off to
// it: the window is going away, and focusing it leaves the user with nothing.
// The refused handoff is what makes Acquire replace the dying process.
func TestRetireRefusesHandoff(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	var focused atomic.Int32
	guard, err := Acquire(func() { focused.Add(1) })
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	t.Cleanup(guard.Close)

	if !handoff(guard.path) {
		t.Fatal("a live instance refused the handoff")
	}
	guard.Retire()
	if handoff(guard.path) {
		t.Fatal("a retiring instance answered the handoff")
	}
	if got := focused.Load(); got != 1 {
		t.Fatalf("focus calls: want 1, got %d", got)
	}
}
