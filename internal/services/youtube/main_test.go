package youtube

import (
	"os"
	"testing"
)

// TestMain points the key-value store at a throwaway directory so tests never
// touch (or lock) the user's real database.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "jota-kv-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", dir)
	os.Setenv("XDG_CONFIG_HOME", dir)
	os.Setenv("APPDATA", dir)

	code := m.Run()

	_ = os.RemoveAll(dir)
	os.Exit(code)
}
