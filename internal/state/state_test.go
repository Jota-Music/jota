package state

import (
	"os"
	"path/filepath"
	"testing"
)

func setup(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	Reset()
	t.Cleanup(Reset)
}

func TestReadWithoutASectionIsEmpty(t *testing.T) {
	setup(t)

	got, err := Read[[]string]("follows")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("follows = %v, want empty", got)
	}
}

func TestWriteAndReadBack(t *testing.T) {
	setup(t)

	type entry struct {
		Id string `json:"id"`
	}
	if err := Write("follows", []entry{{Id: "one"}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := Read[[]entry]("follows")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) != 1 || got[0].Id != "one" {
		t.Fatalf("follows = %v", got)
	}
}

// The document is what the app keeps between launches: what is written has to
// be there after a restart.
func TestTheDocumentSurvivesARestart(t *testing.T) {
	setup(t)

	if err := Write("rooms", []string{"relay|code"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	Close()

	got, err := Read[[]string]("rooms")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(got) != 1 || got[0] != "relay|code" {
		t.Fatalf("rooms = %v", got)
	}
}

// A written section is read back with its value intact.
func TestTheSectionIsReadable(t *testing.T) {
	setup(t)

	if err := Write("follows", map[string]string{"id": "one"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	Close()

	got, err := Read[map[string]string]("follows")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got["id"] != "one" {
		t.Fatalf("follows = %v", got)
	}
}

// A section another build wrote is not this build's to throw away: writing
// one section leaves the others untouched.
func TestAnUnknownSectionIsKept(t *testing.T) {
	setup(t)

	if err := Write("other", map[string]bool{"kept": true}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := Write("follows", map[string]string{"id": "one"}); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := Read[map[string]bool]("other")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !got["kept"] {
		t.Fatalf("other section was not kept: %v", got)
	}
}

// A document that will not decode is kept aside instead of being replaced by
// the next write, so a bad byte never costs the library.
func TestAnUnreadableDocumentIsKeptAside(t *testing.T) {
	setup(t)

	path := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "jota", "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{half"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := Read[[]string]("follows"); err != nil {
		t.Fatalf("read: %v", err)
	}

	if _, err := os.Stat(path + ".bad"); err != nil {
		t.Fatalf("the unreadable document was not kept aside: %v", err)
	}
}
