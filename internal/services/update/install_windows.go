//go:build windows

package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// targetDir stages the new .exe in the install directory. Windows only lets us
// rename a running executable, not replace it, so the new blob must already sit
// on the same volume.
func targetDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func install(staged string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return fmt.Errorf("update: move running exe: %w", err)
	}
	if err := os.Rename(staged, exe); err != nil {
		_ = os.Rename(old, exe) // roll back
		return fmt.Errorf("update: replace exe: %w", err)
	}
	if err := relaunch(exe, old); err != nil {
		return fmt.Errorf("update: installed, reopen Jota (%v)", err)
	}
	return nil
}

// relaunch waits for this process to exit, deletes the old exe and starts the
// new one.
func relaunch(path, old string) error {
	script := fmt.Sprintf(
		"Wait-Process -Id %d -ErrorAction SilentlyContinue; "+
			"Remove-Item '%s' -Force -ErrorAction SilentlyContinue; "+
			"Start-Process -FilePath '%s'",
		os.Getpid(), psQuote(old), psQuote(path))
	cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000008 | 0x00000200, // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
	}
	return cmd.Start()
}

// psQuote escapes a value for a single-quoted PowerShell string (” = ').
func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
