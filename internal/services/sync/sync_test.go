package sync

import "testing"

func TestEndpoint(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"relay.example.com", "wss://relay.example.com/ws?role=host&room=abc"},
		{"https://relay.example.com", "wss://relay.example.com/ws?role=host&room=abc"},
		{"http://localhost:8080", "ws://localhost:8080/ws?role=host&room=abc"},
		{"wss://relay.example.com/custom", "wss://relay.example.com/custom?role=host&room=abc"},
	}
	for _, c := range cases {
		got, err := endpoint(c.raw, "abc", "host")
		if err != nil {
			t.Fatalf("endpoint(%q): %v", c.raw, err)
		}
		if got != c.want {
			t.Errorf("endpoint(%q) = %q, want %q", c.raw, got, c.want)
		}
	}

	if _, err := endpoint("", "abc", "host"); err == nil {
		t.Error("empty url should fail")
	}
	if _, err := endpoint("relay.example.com", "", "host"); err == nil {
		t.Error("empty room should fail")
	}
	if _, err := endpoint("ftp://relay.example.com", "abc", "host"); err == nil {
		t.Error("unsupported scheme should fail")
	}
}

func TestHealthURL(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"relay.example.com", "https://relay.example.com/healthz"},
		{"https://relay.example.com", "https://relay.example.com/healthz"},
		{"wss://relay.example.com/custom", "https://relay.example.com/healthz"},
		{"http://localhost:8080", "http://localhost:8080/healthz"},
	}
	for _, c := range cases {
		got, err := healthURL(c.raw)
		if err != nil {
			t.Fatalf("healthURL(%q): %v", c.raw, err)
		}
		if got != c.want {
			t.Errorf("healthURL(%q) = %q, want %q", c.raw, got, c.want)
		}
	}

	if _, err := healthURL(""); err == nil {
		t.Error("empty url should fail")
	}
}
