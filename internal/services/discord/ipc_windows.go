//go:build windows

package discord

import (
	"fmt"
	"net"
	"os"
)

// ponytail: opens the named pipe with os.OpenFile (blocking handle). Discord
// reads fine from sync handles; if a future Discord moves to async pipes this
// needs a winio/x-sys CreateFile + overlapped wrapper.
type pipe struct {
	*os.File
}

func (p *pipe) LocalAddr() net.Addr  { return nil }
func (p *pipe) RemoteAddr() net.Addr { return nil }
func platformDial() (net.Conn, error) {
	var last error
	for i := 0; i < 10; i++ {
		path := fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i)
		f, err := os.OpenFile(path, os.O_RDWR, 0)
		if err == nil {
			return &pipe{File: f}, nil
		}
		last = err
	}
	return nil, last
}
