package rooms

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConnectTokenRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := NewRelay(nil).Connect(srv.URL, "room", "host", "", "")
	if !errors.Is(err, ErrTokenRequired) {
		t.Fatalf("got %v, want ErrTokenRequired", err)
	}
}

func TestCheckAuthFlag(t *testing.T) {
	cases := []struct {
		body string
		want bool
	}{
		{`{"auth":true}`, true},
		{`{"auth":false}`, false},
		{`ok`, false},
	}
	for _, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(c.body))
		}))
		got, err := NewRelay(nil).Check(srv.URL, "token")
		srv.Close()
		if err != nil {
			t.Fatalf("Check(%q): %v", c.body, err)
		}
		if got != c.want {
			t.Errorf("Check(%q) = %v, want %v", c.body, got, c.want)
		}
	}
}

// A relay behind AUTH_TOKEN answers 401 on /rooms, exactly as it does on /ws,
// which is the only place a token can be judged. Everything but a 401 must
// leave the token unjudged instead of failing the whole check.
func TestCheckToken(t *testing.T) {
	cases := []struct {
		name      string
		auth      bool
		probeCode int
		token     string
		wantAuth  bool
		wantErr   bool
		wantProbe bool
	}{
		{name: "no auth", auth: false, token: "", wantAuth: false},
		{name: "no auth ignores token", auth: false, token: "token", wantAuth: false},
		{name: "missing token", auth: true, token: "", wantAuth: true, wantErr: true},
		{name: "accepted", auth: true, probeCode: 200, token: "token", wantAuth: true, wantProbe: true},
		{name: "rejected", auth: true, probeCode: 401, token: "token", wantAuth: true, wantErr: true, wantProbe: true},
		{name: "unknown endpoint", auth: true, probeCode: 404, token: "token", wantAuth: true, wantProbe: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			probed := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/healthz" {
					probed = true
					if r.URL.Path != "/rooms/"+probeRoom {
						t.Errorf("probe path = %q, want /rooms/%s", r.URL.Path, probeRoom)
					}
					if got := r.Header.Get("Authorization"); got != "Bearer "+c.token {
						t.Errorf("probe auth = %q, want %q", got, "Bearer "+c.token)
					}
					w.WriteHeader(c.probeCode)
					_, _ = w.Write([]byte(`{"active":false}`))
					return
				}
				_, _ = w.Write([]byte(fmt.Sprintf(`{"auth":%t}`, c.auth)))
			}))
			defer srv.Close()

			got, err := NewRelay(nil).Check(srv.URL, c.token)
			if (err != nil) != c.wantErr {
				t.Fatalf("Check() error = %v, wantErr %v", err, c.wantErr)
			}
			if c.wantErr && !errors.Is(err, ErrTokenRequired) {
				t.Fatalf("Check() error = %v, want ErrTokenRequired", err)
			}
			if got != c.wantAuth {
				t.Errorf("Check() = %v, want %v", got, c.wantAuth)
			}
			if probed != c.wantProbe {
				t.Errorf("probed /rooms = %v, want %v", probed, c.wantProbe)
			}
		})
	}
}

func TestCheckUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	if _, err := NewRelay(nil).Check(srv.URL, ""); err == nil {
		t.Fatal("a 502 relay should fail the check")
	}
}

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
