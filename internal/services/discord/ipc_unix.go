//go:build !windows

package discord

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// baseDirs returns candidate socket base directories in priority order:
// XDG_RUNTIME_DIR (Linux/Flatpak), /run/user/<uid> fallback, then the OS temp
// dir (honors $TMPDIR on macOS) and /tmp as final fallback.
func baseDirs() []string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	add(os.Getenv("XDG_RUNTIME_DIR"))
	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		add(filepath.Join("/run/user", strconv.Itoa(os.Getuid())))
	}
	add(os.TempDir())
	add("/tmp")
	return out
}

func platformDial() (net.Conn, error) {
	var last error
	for _, base := range baseDirs() {
		dirs := []string{base}
		for _, sub := range []string{"snap.discord", filepath.Join("app", "com.discordapp.Discord")} {
			dirs = append(dirs, filepath.Join(base, sub))
		}
		for _, dir := range dirs {
			for i := 0; i < 10; i++ {
				conn, err := net.DialTimeout("unix", filepath.Join(dir, "discord-ipc-"+strconv.Itoa(i)), 3*time.Second)
				if err == nil {
					return conn, nil
				}
				last = err
			}
		}
	}
	return nil, last
}
