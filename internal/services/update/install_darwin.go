//go:build darwin

package update

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// targetDir stages the .zip on the same volume as the .app bundle so both the
// extraction and the final swap are plain renames.
func targetDir() (string, error) {
	bundle, err := bundlePath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(bundle), nil
}

func install(staged string) error {
	bundle, err := bundlePath()
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp(filepath.Dir(bundle), ".jota-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	if out, err := exec.Command("ditto", "-x", "-k", staged, dir).CombinedOutput(); err != nil {
		return fmt.Errorf("update: unpack: %w: %s", err, out)
	}
	newApp, err := appIn(dir)
	if err != nil {
		return err
	}

	// Move the running bundle aside instead of deleting it: macOS keeps the
	// process alive against the renamed path, and a straight RemoveAll can fail
	// while WebKit helpers still hold files inside it.
	old := bundle + ".old"
	_ = os.RemoveAll(old)
	if err := os.Rename(bundle, old); err != nil {
		return fmt.Errorf("update: move current app: %w", err)
	}
	if err := os.Rename(newApp, bundle); err != nil {
		_ = os.Rename(old, bundle) // roll back
		return fmt.Errorf("update: replace bundle: %w", err)
	}
	if err := relaunch(bundle, old); err != nil {
		return fmt.Errorf("update: installed, reopen Jota (%v)", err)
	}
	return nil
}

// bundlePath walks up from the running executable to the enclosing .app.
func bundlePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	for p := filepath.Dir(exe); p != "/" && p != "."; p = filepath.Dir(p) {
		if strings.HasSuffix(p, ".app") {
			return p, nil
		}
	}
	return "", errors.New("update: not running from an .app bundle")
}

func appIn(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".app") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", errors.New("update: archive has no .app")
}

// relaunch waits for this process to exit, removes the old bundle and opens the
// new one.
func relaunch(bundle, old string) error {
	script := `while kill -0 "$1" 2>/dev/null; do sleep 0.2; done; rm -rf "$2"; open "$3"`
	cmd := exec.Command("sh", "-c", script, "sh", fmt.Sprint(os.Getpid()), old, bundle)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Start()
}
