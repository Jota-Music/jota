package youtube

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func resetAuth(t *testing.T) {
	t.Helper()
	t.Setenv("YOUTUBE_COOKIES", "")

	authMu.Lock()
	origCookie, origLoaded := authCookie, authLoaded
	authCookie, authLoaded = "", false
	authMu.Unlock()
	_ = authBucket.Delete(authKey)

	t.Cleanup(func() {
		authMu.Lock()
		authCookie, authLoaded = origCookie, origLoaded
		authMu.Unlock()
		_ = authBucket.Delete(authKey)
	})
}

func TestSetCookiesValidatesSapisid(t *testing.T) {
	resetAuth(t)

	if err := SetCookies(""); err == nil {
		t.Fatal("empty cookies should fail")
	}
	if err := SetCookies("SID=abc; HSID=def"); err == nil {
		t.Fatal("cookies without SAPISID should fail")
	}
	if err := SetCookies("SID=abc; SAPISID=sapisid123"); err != nil {
		t.Fatalf("valid cookies: %v", err)
	}
	if !HasAuth() {
		t.Fatal("expected to be signed in")
	}

	if err := ClearCookies(); err != nil {
		t.Fatalf("ClearCookies: %v", err)
	}
	if HasAuth() {
		t.Fatal("expected to be signed out")
	}
}

func TestNormalizeCookies(t *testing.T) {
	header := "SID=a; SAPISID=b; HSID=c"
	if got := normalizeCookies(header); got != header {
		t.Fatalf("header = %q, want %q", got, header)
	}

	netscape := "# Netscape HTTP Cookie File\n" +
		".youtube.com\tTRUE\t/\tTRUE\t0\tSAPISID\tsapisid123\n" +
		".youtube.com\tTRUE\t/\tFALSE\t0\tSID\tsidval\n" +
		"#HttpOnly_.google.com\tTRUE\t/\tTRUE\t0\tHSID\thsidval\n" +
		".example.com\tTRUE\t/\tTRUE\t0\tOTHER\tnope\n"
	got := normalizeCookies(netscape)
	for _, want := range []string{"SAPISID=sapisid123", "SID=sidval", "HSID=hsidval"} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q missing from %q", want, got)
		}
	}
	if strings.Contains(got, "OTHER") {
		t.Fatalf("non-youtube domain kept: %q", got)
	}
}

func TestSetCookiesAcceptsCookiesTxt(t *testing.T) {
	resetAuth(t)

	txt := ".youtube.com\tTRUE\t/\tTRUE\t0\tSAPISID\tsapisid123\n" +
		".youtube.com\tTRUE\t/\tFALSE\t0\tSID\tabc\n"
	if err := SetCookies(txt); err != nil {
		t.Fatalf("SetCookies(cookies.txt): %v", err)
	}
	if !HasAuth() {
		t.Fatal("expected to be signed in")
	}
}

func TestSetCookiesAcceptsSecure3Papisid(t *testing.T) {
	resetAuth(t)

	if err := SetCookies("SID=abc; __Secure-3PAPISID=val3p"); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}
	if !HasAuth() {
		t.Fatal("expected to be signed in")
	}
	if h := authHeaders("https://www.youtube.com"); h == nil {
		t.Fatal("expected auth headers from __Secure-3PAPISID")
	}
}

func TestAuthHeadersSignature(t *testing.T) {
	resetAuth(t)
	if err := SetCookies("SAPISID=sapisid123; SID=abc"); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}

	h := authHeaders("https://www.youtube.com")
	if h == nil {
		t.Fatal("expected auth headers")
	}
	if !strings.HasPrefix(h["Authorization"], "SAPISIDHASH ") {
		t.Fatalf("Authorization = %q", h["Authorization"])
	}
	sig := strings.TrimPrefix(h["Authorization"], "SAPISIDHASH ")
	ts, hash, ok := strings.Cut(sig, "_")
	if !ok || ts == "" || len(hash) != 40 {
		t.Fatalf("malformed signature %q", sig)
	}
	if h["X-Origin"] != "https://www.youtube.com" {
		t.Fatalf("X-Origin = %q", h["X-Origin"])
	}
	if h["Cookie"] == "" {
		t.Fatal("missing Cookie header")
	}
}

// TestAgeRestrictedWithCookiesIntegration is the gate for the login path: with
// account cookies set, an age-restricted video that fails unauthenticated must
// resolve. Skipped unless YOUTUBE_COOKIES is set.
func TestAgeRestrictedWithCookiesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: reaches YouTube")
	}
	if !HasAuth() {
		t.Skip("set YOUTUBE_COOKIES to run the login gate")
	}

	raw, err := playerStreamURL(preferredClient, "qnM1e_9uwEE")
	if err != nil {
		t.Fatalf("age-restricted video failed while signed in: %v", err)
	}
	if raw.URL == "" {
		t.Fatal("empty stream URL")
	}
}

func TestAuthHeadersAbsentWhenSignedOut(t *testing.T) {
	resetAuth(t)
	if h := authHeaders("https://www.youtube.com"); h != nil {
		t.Fatalf("expected nil headers, got %v", h)
	}
}

func TestDoRequestSignsWhenSignedIn(t *testing.T) {
	resetAuth(t)
	if err := SetCookies("SAPISID=sapisid123"); err != nil {
		t.Fatalf("SetCookies: %v", err)
	}
	seedVisitor(t, "vis-token")

	var gotAuth, gotOrigin string
	withInnertubeTransport(t, func(r *http.Request) (*http.Response, error) {
		gotAuth = r.Header.Get("Authorization")
		gotOrigin = r.Header.Get("X-Origin")
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})

	if _, err := retryRequest(webClient(), "https://www.youtube.com/youtubei/v1/player", map[string]any{}, 1); err != nil {
		t.Fatalf("retryRequest: %v", err)
	}
	if !strings.HasPrefix(gotAuth, "SAPISIDHASH ") {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotOrigin != "https://www.youtube.com" {
		t.Fatalf("X-Origin = %q", gotOrigin)
	}
}
