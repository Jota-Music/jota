// Package single keeps one jota per session. A second launch focuses the running
// window instead of starting a rival, and an instance that holds the lock without
// answering is replaced instead of blocking every later launch forever.
//
// Wails does not ship that: its single instance goes over the session D-Bus with
// no timeout, so an instance wedged before its window exists (a startup blocked
// on the network) makes each following launch hang silently, with no window and
// no log line. Here the handshake is bounded, and a wedged owner is killed.
package single

import (
	"errors"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// ErrHandedOff reports that another instance owns the session and was asked to
// show its window; the caller must exit without starting up.
var ErrHandedOff = errors.New("another instance is running")

const (
	// dialTimeout bounds the handshake and the wait for its reply: an instance
	// that cannot answer this fast is treated as wedged.
	dialTimeout = 500 * time.Millisecond
	// termTimeout then killTimeout bound the wait for a replaced instance to
	// actually release the store lock.
	termTimeout  = 2 * time.Second
	killTimeout  = time.Second
	focusMessage = "focus\n"
	okReply      = "ok\n"
	pidSuffix    = ".pid"
)

// Guard owns the per-session lock.
type Guard struct {
	path  string
	ln    net.Listener
	focus func()
	once  sync.Once
	// retiring is set when this instance starts closing. From then on it refuses
	// the handoff, so a launch arriving mid-teardown replaces it instead of being
	// focused at a process that is about to disappear.
	retiring atomic.Bool
}

// Acquire takes the per-session lock. It returns ErrHandedOff when a running
// instance answered and was asked to focus; otherwise it owns the lock, having
// replaced an owner that held it without answering.
func Acquire(focus func()) (*Guard, error) {
	path, err := socketPath()
	if err != nil {
		return nil, err
	}
	if handoff(path) {
		return nil, ErrHandedOff
	}
	replace(path)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path+pidSuffix, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		_ = ln.Close()
		return nil, err
	}

	g := &Guard{path: path, ln: ln, focus: focus}
	go g.serve()
	return g, nil
}

// Retire stops answering handoffs. The socket and the pid file stay, so a launch
// arriving while this instance tears down kills it and takes over rather than
// focusing a window that will not be there.
func (g *Guard) Retire() {
	g.retiring.Store(true)
	_ = g.ln.Close()
}

// Close releases the socket and the pid file.
func (g *Guard) Close() {
	g.once.Do(func() {
		_ = g.ln.Close()
		_ = os.Remove(g.path)
		_ = os.Remove(g.path + pidSuffix)
	})
}

func (g *Guard) serve() {
	for {
		conn, err := g.ln.Accept()
		if err != nil {
			return
		}
		go g.answer(conn)
	}
}

// answer replies before touching the window: the caller is waiting on this reply,
// while focusing goes through the UI event loop, which may be busy.
func (g *Guard) answer(conn net.Conn) {
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(dialTimeout))
	message := make([]byte, len(focusMessage))
	if _, err := io.ReadFull(conn, message); err != nil || string(message) != focusMessage {
		return
	}
	// Refuse once this instance started closing: replying "ok" sends the caller
	// away to wait for a window that is already going. A refused handoff makes
	// Acquire replace this process instead.
	if g.retiring.Load() {
		return
	}
	_, _ = conn.Write([]byte(okReply))
	if g.focus != nil {
		g.focus()
	}
}

// handoff asks the instance owning the socket to show its window. It reports true
// only when that instance replied: a wedged one holds the socket without
// answering, and waiting on it is the hang this package exists to avoid.
func handoff(path string) bool {
	conn, err := net.DialTimeout("unix", path, dialTimeout)
	if err != nil {
		return false
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(focusMessage)); err != nil {
		return false
	}
	_ = conn.SetReadDeadline(time.Now().Add(dialTimeout))
	reply := make([]byte, len(okReply))
	if _, err := io.ReadFull(conn, reply); err != nil {
		return false
	}
	return string(reply) == okReply
}

// replace clears the socket of an owner that is still alive but never answered.
// It is killed rather than ignored: it holds the store lock, so taking over while
// it lives would leave this instance running without persistence.
//
// ponytail: pid reuse within one session is taken at face value; compare the
// process start time against the pid file's mtime if that ever misfires.
func replace(path string) {
	pid := readPid(path + pidSuffix)
	if pid <= 0 || pid == os.Getpid() || !alive(pid) {
		return
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return
	}

	log.Printf("single: instance %d stopped answering, replacing it", pid)
	_ = p.Signal(syscall.SIGTERM)
	if gone(pid, termTimeout) {
		return
	}
	_ = p.Kill()
	_ = gone(pid, killTimeout)
}

// gone reports whether pid disappeared within d.
func gone(pid int, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for !time.Now().After(deadline) {
		if !alive(pid) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return !alive(pid)
}

// alive reports whether the process exists. Signal 0 checks existence and
// permission only; it is never delivered.
func alive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

func readPid(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// socketPath is the per-session control socket. XDG_RUNTIME_DIR is per user and
// wiped at logout; the cache dir is the per-user fallback for the platforms
// without it (this only ever runs on Linux).
func socketPath() (string, error) {
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		if _, err := os.Stat(dir); err == nil {
			return filepath.Join(dir, "jota.sock"), nil
		}
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "jota.sock"), nil
}
