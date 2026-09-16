//go:build linux

package update

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// appImagePath resolves $APPIMAGE, following symlinks so an update replaces the
// real file and not the symlink that points at it.
func appImagePath() (string, error) {
	p := os.Getenv("APPIMAGE")
	if p == "" {
		return "", errors.New("update: not running from an AppImage")
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved, nil
	}
	return p, nil
}

// targetDir stages the download next to the running AppImage so the final
// swap is a same-filesystem rename.
func targetDir() (string, error) {
	target, err := appImagePath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(target), nil
}

func install(staged string) error {
	target, err := appImagePath()
	if err != nil {
		return err
	}
	if err := os.Chmod(staged, 0o755); err != nil {
		return err
	}
	if err := os.Rename(staged, target); err != nil {
		return fmt.Errorf("update: replace AppImage: %w", err)
	}
	if err := relaunch(target); err != nil {
		return fmt.Errorf("update: installed, reopen Jota (%v)", err)
	}
	return nil
}

// relaunch waits for this process to exit, then execs the new AppImage.
// Starting it while this one is alive would only focus the running instance.
// It is a variable so tests can stub it out.
var relaunch = func(path string) error {
	script := `while kill -0 "$1" 2>/dev/null; do sleep 0.2; done; exec "$2"`
	cmd := exec.Command("sh", "-c", script, "sh", fmt.Sprint(os.Getpid()), path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
