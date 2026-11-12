package spotify

import (
	"strings"
	"testing"
)

func TestNeedsRevalidate(t *testing.T) {
	cases := []struct {
		name     string
		cached   string
		current  string
		expected bool
	}{
		{"same revision", "abc", "abc", false},
		{"changed revision", "abc", "def", true},
		{"unknown current revision", "abc", "", false},
		{"missing cached revision", "", "def", true},
		{"both unknown", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := needsRevalidate(tc.cached, tc.current); got != tc.expected {
				t.Fatalf("needsRevalidate(%q, %q) = %v, want %v", tc.cached, tc.current, got, tc.expected)
			}
		})
	}
}

// Two accounts on one device are two libraries: the name carries the account
// so one does not answer for the other.
func TestTheLibraryIsCachedPerAccount(t *testing.T) {
	if libraryName("first") == libraryName("second") {
		t.Fatalf("both accounts share the cache name %q", libraryName("first"))
	}
	if !strings.Contains(libraryName("first"), "first") {
		t.Fatalf("the name does not carry the account: %q", libraryName("first"))
	}
}
