package app

import (
	"errors"
	"slices"
	"testing"

	"github.com/Jota-Music/jota/internal/services/youtube"
)

// jars answers the browser readers with a fixed set, so a test never touches this
// machine's browsers.
func jars(t *testing.T, sessions map[string]string) {
	t.Helper()
	names := make([]string, 0, len(sessions))
	for name := range sessions {
		names = append(names, name)
	}
	slices.Sort(names)
	prevBrowsers, prevCookies := cookiesFromBrowsers, browserCookies
	cookiesFromBrowsers = func() []string { return names }
	browserCookies = func(name string) (string, error) {
		if raw, ok := sessions[name]; ok {
			return raw, nil
		}
		return "", errors.New("no jar for " + name)
	}
	t.Cleanup(func() { cookiesFromBrowsers, browserCookies = prevBrowsers, prevCookies })
}

const fullSession = "SAPISID=sapisid123; __Secure-3PSID=sid3p"

// The clear is local, so a launch with no cookies has to ask the document for the
// session instead of staying signed out until someone imports it by hand.
func TestLaunchRestoresTheSessionFromTheDocument(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncAuto(false); err != nil {
		t.Fatalf("disable autosave: %v", err)
	}
	t.Setenv("YOUTUBE_COOKIES", "")
	_ = youtube.ClearCookies()
	t.Cleanup(func() { _ = youtube.ClearCookies() })

	blob, err := encryptWithPass("shared", fullSession)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if err := a.SetSessionPass("shared"); err != nil {
		t.Fatalf("set pass: %v", err)
	}
	if err := stageBlob("youtube", blob); err != nil {
		t.Fatalf("stage: %v", err)
	}

	refreshYouTubeSession()

	if got := youtube.Cookies(); got != fullSession {
		t.Fatalf("restored session = %q, want %q", got, fullSession)
	}
}

// The remembered browser is the jar Google refreshes the session in, so a launch
// takes its copy without asking YouTube anything.
func TestLaunchRefreshesFromTheRememberedBrowser(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncAuto(false); err != nil {
		t.Fatalf("disable autosave: %v", err)
	}
	t.Setenv("YOUTUBE_COOKIES", "")
	_ = youtube.ClearCookies()
	t.Cleanup(func() { _ = youtube.ClearCookies() })

	const stale = "SAPISID=sapisid123; __Secure-3PSID=stale"
	const fresh = "SAPISID=sapisid123; __Secure-3PSID=fresh"
	if err := youtube.SetCookiesFrom(stale, "Firefox"); err != nil {
		t.Fatalf("set cookies: %v", err)
	}
	jars(t, map[string]string{"Firefox": fresh, "Chrome": "SAPISID=other; __Secure-3PSID=other"})

	refreshYouTubeSession()

	if got := youtube.Cookies(); got != fresh {
		t.Fatalf("session after a launch = %q, want %q", got, fresh)
	}
	if got := youtube.SessionBrowser(); got != "Firefox" {
		t.Fatalf("remembered browser = %q, want Firefox", got)
	}
}

// A device that never had a session is not signed in behind the user's back.
func TestLaunchReadsNoJarWithoutASession(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncAuto(false); err != nil {
		t.Fatalf("disable autosave: %v", err)
	}
	t.Setenv("YOUTUBE_COOKIES", "")
	_ = youtube.ClearCookies()
	t.Cleanup(func() { _ = youtube.ClearCookies() })

	read := false
	prevBrowsers := cookiesFromBrowsers
	cookiesFromBrowsers = func() []string {
		read = true
		return []string{"Firefox"}
	}
	t.Cleanup(func() { cookiesFromBrowsers = prevBrowsers })

	refreshYouTubeSession()

	if read {
		t.Fatal("a device with no session read the browser jar")
	}
	if got := youtube.Cookies(); got != "" {
		t.Fatalf("a session was created from the browser: %q", got)
	}
}

// A set that only signs requests never replaces one that opens the account's own
// lists, not even when the jar hands it over.
func TestPartialJarDoesNotReplaceAFullSession(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncAuto(false); err != nil {
		t.Fatalf("disable autosave: %v", err)
	}
	t.Setenv("YOUTUBE_COOKIES", "")
	_ = youtube.ClearCookies()
	t.Cleanup(func() { _ = youtube.ClearCookies() })

	if err := youtube.SetCookiesFrom(fullSession, "Firefox"); err != nil {
		t.Fatalf("set cookies: %v", err)
	}
	jars(t, map[string]string{"Firefox": "SAPISID=sapisid123"})

	refreshYouTubeSession()

	if got := youtube.Cookies(); got != fullSession {
		t.Fatalf("a partial jar replaced a full session: %q", got)
	}
}

// The jar the session came from is refreshed by the browser itself, so it is
// asked before any other, account or not.
func TestRecoveryAsksTheRememberedBrowserFirst(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncAuto(false); err != nil {
		t.Fatalf("disable autosave: %v", err)
	}
	t.Setenv("YOUTUBE_COOKIES", "")
	_ = youtube.ClearCookies()
	t.Cleanup(func() { _ = youtube.ClearCookies() })

	const stale = "SAPISID=stale; __Secure-3PSID=stale"
	const rotated = "SAPISID=rotated; __Secure-3PSID=rotated"
	const elsewhere = "SAPISID=stale; __Secure-3PSID=elsewhere"
	if err := youtube.SetCookiesFrom(stale, "Chrome"); err != nil {
		t.Fatalf("set cookies: %v", err)
	}
	jars(t, map[string]string{"Chrome": rotated, "Firefox": elsewhere})

	if !recoverYouTubeSession() {
		t.Fatal("no session was recovered")
	}
	if got := youtube.Cookies(); got != rotated {
		t.Fatalf("session after recovery = %q, want the remembered browser's %q", got, rotated)
	}
}

// Without a remembered browser, only a jar that continues the same account is
// taken: signing the device in as someone else costs more than an import.
func TestRecoveryOnlyTakesTheSameAccount(t *testing.T) {
	a := syncApp(t)
	if err := a.SetSyncAuto(false); err != nil {
		t.Fatalf("disable autosave: %v", err)
	}
	t.Setenv("YOUTUBE_COOKIES", "")
	_ = youtube.ClearCookies()
	t.Cleanup(func() { _ = youtube.ClearCookies() })

	const stale = "SAPISID=stale; __Secure-3PSID=stale"
	const other = "SAPISID=other; __Secure-3PSID=other"
	const renewed = "SAPISID=stale; __Secure-3PSID=renewed"
	if err := youtube.SetCookies(stale); err != nil {
		t.Fatalf("set cookies: %v", err)
	}
	jars(t, map[string]string{"Chrome": other, "Firefox": renewed})

	if !recoverYouTubeSession() {
		t.Fatal("the jar continuing the account was not taken")
	}
	if got := youtube.Cookies(); got != renewed {
		t.Fatalf("session after recovery = %q, want %q", got, renewed)
	}

	jars(t, map[string]string{"Chrome": other})
	if recoverYouTubeSession() {
		t.Fatalf("another account's jar was taken: %q", youtube.Cookies())
	}
	if got := youtube.Cookies(); got != renewed {
		t.Fatalf("a failed recovery changed the session: %q", got)
	}
}
