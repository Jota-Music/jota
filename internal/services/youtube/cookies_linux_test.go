//go:build linux

package youtube

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// testSecret is the password the test jars are sealed with: the one Chromium
// uses when this desktop keeps no keyring, so no keyring has to answer.
const testSecret = "peanuts"

// storedCookie is one row of a jar: plaintext value or an encrypted blob.
type storedCookie struct {
	host  string
	name  string
	value string
	blob  []byte
}

// jarPath is where a Chromium browser keeps one profile's cookies.
func jarPath(config, browser, profile string) string {
	return filepath.Join(config, browser, profile, "Network", "Cookies")
}

// simulate points os.UserConfigDir and os.UserHomeDir at temp trees, so the
// discovery walks a machine this test built instead of the real one.
func simulate(t *testing.T) (config, home string) {
	t.Helper()
	config, home = t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("HOME", home)
	return config, home
}

// quietKeyring keeps a jar read off this machine's keyring: the lookup would
// spawn secret-tool and possibly wait for an unlock prompt, and the test wants
// the fixed password, not the user's.
func quietKeyring(t *testing.T, names ...string) {
	t.Helper()
	type entry struct {
		secrets []string
		known   bool
	}
	before := map[string]entry{}
	keyringMu.Lock()
	for _, name := range names {
		before[name] = entry{keyringSecrets[name], keyringed[name]}
		keyringSecrets[name] = nil
		keyringed[name] = true
	}
	keyringMu.Unlock()
	t.Cleanup(func() {
		keyringMu.Lock()
		for name, old := range before {
			keyringSecrets[name], keyringed[name] = old.secrets, old.known
		}
		keyringMu.Unlock()
	})
}

// writeChromiumJar writes a jar the way Chromium lays it out: one cookies table
// and a meta row carrying the jar format version.
func writeChromiumJar(t *testing.T, path string, version int, cookies []storedCookie) {
	t.Helper()
	open := jarFile(t, path)
	defer open.Close()

	for _, stmt := range []string{
		`CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT)`,
		`CREATE TABLE cookies (host_key TEXT, name TEXT, value TEXT, encrypted_value BLOB)`,
	} {
		if _, err := open.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := open.Exec(`INSERT INTO meta (key, value) VALUES ('version', ?)`, strconv.Itoa(version)); err != nil {
		t.Fatal(err)
	}
	for _, c := range cookies {
		if _, err := open.Exec(`INSERT INTO cookies (host_key, name, value, encrypted_value) VALUES (?, ?, ?, ?)`,
			c.host, c.name, c.value, c.blob); err != nil {
			t.Fatal(err)
		}
	}
}

// writeFirefoxJar writes a Firefox jar: cookies in the clear, no version.
func writeFirefoxJar(t *testing.T, path string, cookies []storedCookie) {
	t.Helper()
	open := jarFile(t, path)
	defer open.Close()

	if _, err := open.Exec(`CREATE TABLE moz_cookies (host TEXT, name TEXT, value TEXT)`); err != nil {
		t.Fatal(err)
	}
	for _, c := range cookies {
		if _, err := open.Exec(`INSERT INTO moz_cookies (host, name, value) VALUES (?, ?, ?)`,
			c.host, c.name, c.value); err != nil {
			t.Fatal(err)
		}
	}
}

// jarFile opens the file at path as SQLite, creating the directories first.
func jarFile(t *testing.T, path string) *sql.DB {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

// boundValue is the plaintext Chromium seals since jar version 24: the SHA-256
// of the cookie's host, then the value, so the blob cannot be replayed on
// another site.
func boundValue(host, value string) []byte {
	sum := sha256.Sum256([]byte(host))
	return append(sum[:], value...)
}

func TestCookiesFromBrowser(t *testing.T) {
	cases := []struct {
		name    string
		version int
		scheme  string
		plain   []byte
	}{
		{"version 19 keeps the plain value", 19, "v10", []byte("sapisid-value")},
		{"version 27 strips the host prefix", 27, "v10", boundValue(".youtube.com", "sapisid-value")},
		{"v11 is the same scheme", 27, "v11", boundValue(".youtube.com", "sapisid-value")},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			quietKeyring(t, "Brave")
			config, _ := simulate(t)
			session := []byte("3psid-value")
			if c.version >= 24 {
				session = boundValue(".youtube.com", "3psid-value")
			}
			writeChromiumJar(t, jarPath(config, "BraveSoftware/Brave-Browser", "Default"), c.version, []storedCookie{
				{host: ".youtube.com", name: "SAPISID", blob: seal(t, c.scheme, testSecret, c.plain)},
				{host: ".youtube.com", name: "__Secure-3PSID", blob: seal(t, c.scheme, testSecret, session)},
			})

			raw, err := CookiesFromBrowser("Brave")
			if err != nil {
				t.Fatalf("CookiesFromBrowser: %v", err)
			}
			if got := cookieValue(raw, "SAPISID"); got != "sapisid-value" {
				t.Fatalf("SAPISID = %q, want sapisid-value", got)
			}
			if got := sessionValue(raw); got != "3psid-value" {
				t.Fatalf("session cookie = %q, want 3psid-value", got)
			}
		})
	}
}

func TestCookiesFromBrowserKeepsOneCopyPerName(t *testing.T) {
	quietKeyring(t, "Brave")
	config, _ := simulate(t)
	writeChromiumJar(t, jarPath(config, "BraveSoftware/Brave-Browser", "Default"), 27, []storedCookie{
		{host: ".youtube.com", name: "SAPISID", blob: seal(t, "v10", testSecret, boundValue(".youtube.com", "youtube-copy"))},
		{host: ".google.com", name: "SAPISID", blob: seal(t, "v10", testSecret, boundValue(".google.com", "google-copy"))},
		{host: ".youtube.com", name: "__Secure-3PSID", blob: seal(t, "v10", testSecret, boundValue(".youtube.com", "3psid"))},
	})

	raw, err := CookiesFromBrowser("Brave")
	if err != nil {
		t.Fatalf("CookiesFromBrowser: %v", err)
	}
	if got := strings.Count(raw, "SAPISID="); got != 1 {
		t.Fatalf("SAPISID appears %d times in %q, want once", got, raw)
	}
	if got := cookieValue(raw, "SAPISID"); got != "youtube-copy" {
		t.Fatalf("SAPISID = %q, want the youtube.com copy", got)
	}
}

func TestCookiesFromBrowserSkipsCorruptRow(t *testing.T) {
	quietKeyring(t, "Chrome")
	config, _ := simulate(t)
	writeChromiumJar(t, jarPath(config, "google-chrome", "Default"), 27, []storedCookie{
		{host: ".youtube.com", name: "SAPISID", blob: seal(t, "v10", testSecret, boundValue(".youtube.com", "sapisid"))},
		{host: ".youtube.com", name: "PREF", blob: []byte("v10broken")},
		{host: ".youtube.com", name: "SSID", blob: seal(t, "v10", testSecret, boundValue(".youtube.com", "ssid"))},
	})

	raw, err := CookiesFromBrowser("Chrome")
	if err != nil {
		t.Fatalf("CookiesFromBrowser: %v", err)
	}
	if got := cookieValue(raw, "SSID"); got != "ssid" {
		t.Fatalf("SSID = %q, want ssid: a corrupt row cost the session", got)
	}
}

func TestCookiesFromBrowserKeyringLocked(t *testing.T) {
	quietKeyring(t, "Vivaldi")
	config, _ := simulate(t)
	writeChromiumJar(t, jarPath(config, "vivaldi", "Default"), 27, []storedCookie{
		{host: ".youtube.com", name: "SAPISID", blob: seal(t, "v10", "unknown-password", boundValue(".youtube.com", "sapisid"))},
	})

	if _, err := CookiesFromBrowser("Vivaldi"); !errors.Is(err, errKeyring) {
		t.Fatalf("CookiesFromBrowser = %v, want errKeyring", err)
	}
}

func TestCookiesFromBrowserPartialJar(t *testing.T) {
	quietKeyring(t, "Brave")
	config, _ := simulate(t)
	writeChromiumJar(t, jarPath(config, "BraveSoftware/Brave-Browser", "Default"), 27, []storedCookie{
		{host: ".youtube.com", name: "SAPISID", blob: seal(t, "v10", testSecret, boundValue(".youtube.com", "sapisid"))},
	})

	raw, err := CookiesFromBrowser("Brave")
	if err != nil {
		t.Fatalf("CookiesFromBrowser: %v", err)
	}
	if got := sessionValue(raw); got != "" {
		t.Fatalf("session cookie = %q, want none", got)
	}
	if got := sidValue(raw); got != "sapisid" {
		t.Fatalf("SAPISID = %q, want sapisid", got)
	}
}

func TestCookiesFromBrowserNotSignedIn(t *testing.T) {
	quietKeyring(t, "Brave")
	config, _ := simulate(t)
	writeChromiumJar(t, jarPath(config, "BraveSoftware/Brave-Browser", "Default"), 27, []storedCookie{
		{host: ".youtube.com", name: "PREF", value: "hl=en"},
	})

	if _, err := CookiesFromBrowser("Brave"); err == nil {
		t.Fatal("CookiesFromBrowser returned a session for a jar without SAPISID")
	}
}

func TestCookiesFromBrowserUnknownBrowser(t *testing.T) {
	quietKeyring(t, "Firefox")
	simulate(t)

	if _, err := CookiesFromBrowser("Firefox"); err == nil {
		t.Fatal("CookiesFromBrowser returned a session for a browser with no jar")
	}
}

func TestCookiesFromBrowserFirefox(t *testing.T) {
	_, home := simulate(t)
	writeFirefoxJar(t, filepath.Join(home, ".mozilla/firefox/abc.default/cookies.sqlite"), []storedCookie{
		{host: ".youtube.com", name: "SAPISID", value: "firefox-sapisid"},
		{host: ".youtube.com", name: "__Secure-3PSID", value: "firefox-3psid"},
	})

	raw, err := CookiesFromBrowser("Firefox")
	if err != nil {
		t.Fatalf("CookiesFromBrowser: %v", err)
	}
	if got := sessionValue(raw); got != "firefox-3psid" {
		t.Fatalf("session cookie = %q, want firefox-3psid", got)
	}
}

func TestCookieStoreFollowsXDGDataHome(t *testing.T) {
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)

	want := filepath.Join(data, "jota", "cookies.sqlite")
	if got := cookieStore(); got != want {
		t.Fatalf("cookieStore = %q, want %q", got, want)
	}
}

func TestBrowsers(t *testing.T) {
	quietKeyring(t, "Brave", "Chrome")
	config, home := simulate(t)

	// A session cookie makes the jar worth offering: CookiesBrowsers leaves out
	// the ones that never signed in.
	youtube := []storedCookie{
		{host: ".youtube.com", name: "SAPISID", value: "sign-only"},
		{host: ".youtube.com", name: "__Secure-3PSID", value: "session"},
	}

	// Brave keeps two profiles, each with its own jar.
	writeChromiumJar(t, jarPath(config, "BraveSoftware/Brave-Browser", "Default"), 27, youtube)
	writeChromiumJar(t, jarPath(config, "BraveSoftware/Brave-Browser", "Profile 1"), 27, youtube)
	if err := os.MkdirAll(filepath.Join(config, "BraveSoftware/Brave-Browser/System Profile"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Chrome keeps its jar at the profile root on older versions.
	writeChromiumJar(t, filepath.Join(config, "google-chrome", "Default", "Cookies"), 27, youtube)

	writeFirefoxJar(t, filepath.Join(home, "snap/firefox/common/.mozilla/firefox/xyz.default/cookies.sqlite"), youtube)

	got := make([]string, 0, 4)
	for _, b := range Browsers() {
		got = append(got, b.Name+"/"+b.Profile)
	}
	want := []string{"Brave/Default", "Brave/Profile 1", "Chrome/Default", "Firefox/xyz.default"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("Browsers = %v, want %v", got, want)
	}
	if names := CookiesBrowsers(); strings.Join(names, ",") != "Brave,Chrome,Firefox" {
		t.Fatalf("CookiesBrowsers = %v, want one entry per browser", names)
	}
}

func TestBrowsersFindsSandboxedFirefoxFork(t *testing.T) {
	_, home := simulate(t)
	path := filepath.Join(home, ".var/app/app.zen_browser.zen/.zen/abc123.Default (release)/cookies.sqlite")
	writeFirefoxJar(t, path, []storedCookie{{host: ".youtube.com", name: "SAPISID", value: "zen"}})

	for _, b := range Browsers() {
		if b.Name == "Zen" && b.Path == path {
			return
		}
	}
	t.Fatalf("the Flatpak Zen jar was not discovered: %+v", Browsers())
}

func TestCookiesBrowsersSkipsSignOnlyJars(t *testing.T) {
	quietKeyring(t, "Chromium", "Vivaldi")
	config, _ := simulate(t)
	// Chromium only ever visited youtube.com, so its jar can sign requests but
	// opens nothing; Vivaldi's holds the cookies a session needs.
	writeChromiumJar(t, jarPath(config, "chromium", "Default"), 19, []storedCookie{
		{host: ".youtube.com", name: "SAPISID", value: "sign-only"},
	})
	writeChromiumJar(t, jarPath(config, "vivaldi", "Default"), 19, []storedCookie{
		{host: ".youtube.com", name: "SAPISID", value: "sign-only"},
		{host: ".youtube.com", name: "__Secure-3PSID", value: "session"},
	})

	got := CookiesBrowsers()
	if len(got) != 1 || got[0] != "Vivaldi" {
		t.Fatalf("CookiesBrowsers = %v, want [Vivaldi]", got)
	}
}
