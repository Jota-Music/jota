package spotify

import "testing"

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
