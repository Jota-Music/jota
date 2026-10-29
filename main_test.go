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
	st := windowState{X: 100, Y: 50, Width: 2000, Height: 1000}
	clampWindowState(&st, screens)
	if st.Width != 2000 || st.Height != 1000 {
		t.Fatalf("in-bounds geometry must be preserved: %+v", st)
	}
	if st.X != 100 || st.Y != 50 {
		t.Fatalf("position must be preserved: %+v", st)
	}

	oversized := windowState{X: 100, Y: 50, Width: 3000, Height: 1000}
	clampWindowState(&oversized, screens)
	if oversized.Width != 2560 || oversized.Height != 1000 {
		t.Fatalf("oversized geometry not capped: %+v", oversized)
	}
	if oversized.X != 0 {
		t.Fatalf("a capped window must not hang off the screen: %+v", oversized)
	}
}

// A geometry saved on a monitor that is gone, or on a layout that shrank, used to
// restore the window past the edge of the screen: visible in part at best, and
// with no way to tell why.
func TestClampWindowStatePullsOffScreenPositionBack(t *testing.T) {
	screens := []*application.Screen{
		{WorkArea: application.Rect{X: 0, Y: 0, Width: 5120, Height: 1440}},
	}
	st := windowState{X: 3238, Y: -400, Width: 2374, Height: 1440}
	clampWindowState(&st, screens)
	if st.X != 2746 || st.Y != 0 {
		t.Fatalf("off-screen position not pulled inside the work area: %+v", st)
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

func TestWindowStateFileRoundTrip(t *testing.T) {
	old := windowPath
	t.Cleanup(func() { windowPath = old })
	windowPath = filepath.Join(t.TempDir(), "window.json")

	want := windowState{X: 10, Y: 20, Width: 900, Height: 600, Maximised: true}
	writeWindowStateFile(want)
	if got := loadWindowState(); got != want {
		t.Fatalf("geometry not restored: %+v", got)
	}
	if _, err := os.Stat(windowPath + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp file left behind: %v", err)
	}
}

// A damaged or half-written geometry file must fall back to the defaults: the
// window is created from this value, so a panic here is an app that never opens.
func TestWindowStateFileIgnoresGarbage(t *testing.T) {
	old := windowPath
	t.Cleanup(func() { windowPath = old })
	windowPath = filepath.Join(t.TempDir(), "window.json")

	for _, body := range []string{"", "{", "null", `{"Width":"wide"}`} {
		if err := os.WriteFile(windowPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := loadWindowState(); got != (windowState{}) {
			t.Fatalf("garbage %q restored %+v", body, got)
		}
	}

	// A missing file is the first launch.
	if err := os.Remove(windowPath); err != nil {
		t.Fatal(err)
	}
	if got := loadWindowState(); got != (windowState{}) {
		t.Fatalf("missing file restored %+v", got)
	}
}

// The window is built before any database exists: reading the geometry must not
// open the store, which is what used to put Badger on the launch path.
func TestLoadWindowStateDoesNotOpenTheStore(t *testing.T) {
	oldPath, oldDir := windowPath, os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() {
		windowPath = oldPath
		_ = os.Setenv("XDG_CONFIG_HOME", oldDir)
	})
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	windowPath = filepath.Join(dir, "window.json")

	writeWindowStateFile(windowState{X: 1, Y: 2, Width: 800, Height: 600})
	if got := loadWindowState(); got.Width != 800 {
		t.Fatalf("geometry not read from the file: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "jota", "storage")); !os.IsNotExist(err) {
		t.Fatalf("reading the geometry created the store: %v", err)
	}
}
