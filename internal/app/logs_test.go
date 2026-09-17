package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLogsAtPutsRotationFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jota.log")
	if err := os.WriteFile(path+".1", []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readLogsAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old\nnew\n" {
		t.Fatalf("got %q, want %q", got, "old\nnew\n")
	}
}

func TestReadLogsAtWithoutRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jota.log")
	if err := os.WriteFile(path, []byte("only\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := readLogsAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "only\n" {
		t.Fatalf("got %q, want %q", got, "only\n")
	}
}
