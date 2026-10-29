// Package clipboard reads the desktop clipboard without blocking the main loop.
//
// GTK's blocking read (gtk_clipboard_wait_for_text) freezes the whole window when
// the selection owner does not answer, which on a Wayland session is routine: an
// XWayland client has to cross the compositor's selection bridge, and when that
// stalls, the GTK thread waits with it. This package asks asynchronously instead:
// the request only schedules work, and the answer or the timeout decides.
package clipboard

import (
	"errors"
	"sync"
	"time"
)

// ErrUnsupported reports a platform that reads the clipboard another way.
var ErrUnsupported = errors.New("clipboard: not supported on this platform")

// Timeout bounds one read. The answer arrives on the platform's own loop, and a
// clipboard owner that never answers must not hold the caller.
const Timeout = 400 * time.Millisecond

// request asks the platform for the text and hands the answer to deliver, which
// the platform may call later and from its own thread. It is nil where the
// platform has no asynchronous read.
var request func(deliver func(text string))

var (
	mu      sync.Mutex
	pending func(string)
)

// Text reads the clipboard, or reports ErrUnsupported. An empty answer and a
// timeout are the same thing for the caller: nothing worth reading. It never
// blocks the caller for longer than Timeout.
func Text() (string, error) {
	if request == nil {
		return "", ErrUnsupported
	}
	answers := make(chan string, 1)
	go request(func(text string) {
		select {
		case answers <- text:
		default:
		}
	})

	select {
	case text := <-answers:
		return text, nil
	case <-time.After(Timeout):
		return "", nil
	}
}

// deliver is what the platform calls back with, whichever request it answers.
// The first answer wins and later ones are dropped, so a slow owner cannot
// resolve a request that already timed out.
func deliver(text string) {
	mu.Lock()
	fn := pending
	pending = nil
	mu.Unlock()
	if fn != nil {
		fn(text)
	}
}

// ask records the delivery slot before the platform schedules the request.
func ask(deliver func(string)) {
	mu.Lock()
	pending = deliver
	mu.Unlock()
}
