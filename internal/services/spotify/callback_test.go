package spotify

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A stray hit on the callback port must not consume the slot the real browser is
// about to fill, otherwise one foreign request kills the pending login and the
// user waits out the timeout for nothing.
func TestCallbackRejectsForeignState(t *testing.T) {
	c, err := newCallbackServer("expected-state")
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	t.Cleanup(c.stop)

	get := func(query string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		c.handle(w, httptest.NewRequest(http.MethodGet, "/login?"+query, nil))
		return w
	}

	if w := get("code=foreign&state=wrong"); w.Code != http.StatusBadRequest {
		t.Errorf("foreign state: got %d, want %d", w.Code, http.StatusBadRequest)
	}
	if w := get("code=foreign"); w.Code != http.StatusBadRequest {
		t.Errorf("missing state: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	// The real redirect must still be able to land.
	if w := get("code=real-code&state=expected-state"); w.Code != http.StatusOK {
		t.Errorf("matching state: got %d, want %d", w.Code, http.StatusOK)
	}

	code := <-c.result
	if code != "real-code" {
		t.Errorf("delivered code = %q, want %q", code, "real-code")
	}
}

// The consent screen can deny access, which is a real answer rather than a
// stranger: it has to end the wait instead of spinning until the timeout.
func TestCallbackProviderErrorEndsWait(t *testing.T) {
	c, err := newCallbackServer("expected-state")
	if err != nil {
		t.Fatalf("newCallbackServer: %v", err)
	}
	t.Cleanup(c.stop)

	w := httptest.NewRecorder()
	c.handle(w, httptest.NewRequest(http.MethodGet,
		"/login?error=access_denied&state=expected-state", nil))

	if code := <-c.result; code != "" {
		t.Errorf("delivered code = %q, want empty", code)
	}
	if body := w.Body.String(); !strings.Contains(body, "Login failed") {
		t.Errorf("body = %q, want a failure page", body)
	}
}
