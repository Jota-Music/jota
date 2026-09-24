package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestClampWindowStateDropsFullScreenResidue(t *testing.T) {
	screens := []*application.Screen{
		{WorkArea: application.Rect{X: 0, Y: 0, Width: 5120, Height: 1440}},
	}
	st := windowState{X: 0, Y: 0, Width: 5120, Height: 1440, Maximised: false}
	clampWindowState(&st, screens)
	if st.Width != 0 || st.Height != 0 {
		t.Fatalf("full-work-area geometry not dropped: %+v", st)
	}
}

func TestClampWindowStateCapsToScreen(t *testing.T) {
	screens := []*application.Screen{
		{WorkArea: application.Rect{X: 0, Y: 0, Width: 2560, Height: 1440}},
	}
	st := windowState{X: 100, Y: 50, Width: 3000, Height: 1000}
	clampWindowState(&st, screens)
	if st.Width != 2560 || st.Height != 1000 {
		t.Fatalf("oversized geometry not capped: %+v", st)
	}
	if st.X != 100 || st.Y != 50 {
		t.Fatalf("position must be preserved: %+v", st)
	}
}

func TestRotatingWriterCapsFileSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jota.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := &rotatingWriter{path: path, max: 100, f: f}
	chunk := bytes.Repeat([]byte("x"), 80)

	if _, err := w.Write(chunk); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(chunk); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected a rotated backup: %v", err)
	}
	if w.size > 100 {
		t.Fatalf("active log not capped: %d bytes", w.size)
	}
}

func TestSetupLoggingWritesAndReusesLog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	closeLog := setupLogging()
	defer closeLog()

	log.Print("hello-from-test")

	body, err := os.ReadFile(filepath.Join(dir, "jota", "jota.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "hello-from-test") {
		t.Fatalf("log line not written: %q", body)
	}
}
