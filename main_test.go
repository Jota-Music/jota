package main

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
