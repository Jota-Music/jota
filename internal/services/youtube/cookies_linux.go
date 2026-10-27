//go:build linux && !android

package youtube

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Browser is one sign-in-capable browser profile found on this machine. A
// browser with several profiles yields one entry per profile, because each one
// keeps its own cookie jar and only the signed-in one is worth reading.
type Browser struct {
	Name    string
	Profile string
	Path    string
}

// jarKind separates the two jar layouts: the Chromium family, which encrypts its
// values, and the Firefox family, which keeps them in the clear.
type jarKind int

const (
	chromiumKind jarKind = iota
	firefoxKind
)

// jarBase says what a browser root is relative to: the XDG config dir or the
// home dir, where Flatpak apps and the Firefox family live.
type jarBase int

const (
	configBase jarBase = iota
	homeBase
)

// browserRoots is every place worth looking for a profile, relative to its base.
// The native install comes first and the sandboxed copies after it: a Flatpak
// browser keeps its own home under ~/.var/app/<id>, so the same root appears
// again one level deeper, and the Firefox family has its own dot directory.
var browserRoots = []struct {
	name string
	root string
	base jarBase
	kind jarKind
}{
	{"Brave", "BraveSoftware/Brave-Browser", configBase, chromiumKind},
	{"Brave", ".var/app/com.brave.Browser/config/BraveSoftware/Brave-Browser", homeBase, chromiumKind},
	{"Chrome", "google-chrome", configBase, chromiumKind},
	{"Chrome", ".var/app/com.google.Chrome/config/google-chrome", homeBase, chromiumKind},
	{"Chromium", "chromium", configBase, chromiumKind},
	{"Chromium", ".var/app/org.chromium.Chromium/config/chromium", homeBase, chromiumKind},
	{"Edge", "microsoft-edge", configBase, chromiumKind},
	{"Edge", ".var/app/com.microsoft.Edge/config/microsoft-edge", homeBase, chromiumKind},
	{"Vivaldi", "vivaldi", configBase, chromiumKind},
	{"Vivaldi", ".var/app/com.vivaldi.Vivaldi/config/vivaldi", homeBase, chromiumKind},
	{"Firefox", ".mozilla/firefox", homeBase, firefoxKind},
	{"Firefox", "snap/firefox/common/.mozilla/firefox", homeBase, firefoxKind},
	{"Firefox", ".var/app/org.mozilla.firefox/.mozilla/firefox", homeBase, firefoxKind},
	{"Zen", ".zen", homeBase, firefoxKind},
	{"Zen", ".var/app/app.zen_browser.zen/.zen", homeBase, firefoxKind},
	{"LibreWolf", ".librewolf", homeBase, firefoxKind},
	{"LibreWolf", ".var/app/io.gitlab.librewolf-community/.librewolf", homeBase, firefoxKind},
	{"Waterfox", ".waterfox", homeBase, firefoxKind},
	{"Waterfox", ".var/app/net.waterfox.waterfox/.waterfox", homeBase, firefoxKind},
	{"Floorp", ".floorp", homeBase, firefoxKind},
	{"Floorp", ".var/app/one.ablaze.floorp/.floorp", homeBase, firefoxKind},
}

// profileJar returns the cookie file of one profile directory, if it has one.
func profileJar(kind jarKind, dir string) string {
	if kind == firefoxKind {
		return cookieFile(filepath.Join(dir, "cookies.sqlite"))
	}
	return cookieFile(filepath.Join(dir, "Cookies"), filepath.Join(dir, "Network", "Cookies"))
}

// Browsers lists the browser profiles on this machine that hold a cookie jar,
// in a stable order. A browser or profile without a jar is left out, so the UI
// only offers what it can actually read.
func Browsers() []Browser {
	var list []Browser
	config, _ := os.UserConfigDir()
	home, _ := os.UserHomeDir()
	seen := make(map[string]bool)

	for _, b := range browserRoots {
		base := config
		if b.base == homeBase {
			base = home
		}
		root := filepath.Join(base, b.root)
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() || skipProfile(entry.Name()) {
				continue
			}
			path := profileJar(b.kind, filepath.Join(root, entry.Name()))
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			list = append(list, Browser{Name: b.name, Profile: entry.Name(), Path: path})
		}
	}

	slices.SortFunc(list, func(a, b Browser) int {
		if n := strings.Compare(a.Name, b.Name); n != 0 {
			return n
		}
		return strings.Compare(a.Profile, b.Profile)
	})
	return list
}

// skipProfile drops the profile directories that never hold a session: the
// browser's own special ones and the hidden bookkeeping dirs.
func skipProfile(name string) bool {
	switch name {
	case "Guest Profile", "System Profile", "Crash Reports":
		return true
	}
	return strings.HasPrefix(name, ".")
}

// CookiesBrowsers lists the browser names whose jar holds a YouTube session, so
// the UI only offers the ones that have something to import. A jar that merely
// visited youtube.com can sign requests but opens nothing, so it stays out; a
// browser with a session in any of its profiles is listed once.
func CookiesBrowsers() []string {
	var names []string
	for _, b := range Browsers() {
		if slices.Contains(names, b.Name) {
			continue
		}
		rows, _, err := readJar(b.Path)
		if err != nil || !hasSessionRows(rows) {
			continue
		}
		names = append(names, b.Name)
	}
	slices.Sort(names)
	return names
}

// hasSessionRows reports whether a jar carries the cookies a full session needs.
// Their values may still be encrypted, so this looks at the names only; a
// browser that visited youtube.com has them, the partial set the sign-in window
// can read does not.
func hasSessionRows(rows []jarRow) bool {
	for _, row := range rows {
		switch row.name {
		case "__Secure-3PSID", "__Secure-1PSID", "SSID":
			return true
		}
	}
	return false
}

// BrowserAccount is one browser and the account its jar is signed in to. The
// account is what tells two browsers apart when both hold a session, so the
// panel can offer the one the user means instead of an opaque name.
type BrowserAccount struct {
	Name    string
	Account string
}

// BrowserAccounts names the account behind each browser that holds a session.
// A browser whose jar answers nothing keeps an empty account: it is still worth
// importing, it just cannot be labelled.
func BrowserAccounts() []BrowserAccount {
	names := CookiesBrowsers()
	list := make([]BrowserAccount, 0, len(names))
	for _, name := range names {
		account := ""
		if raw, err := CookiesFromBrowser(name); err == nil {
			account, _ = accountName(raw)
		}
		list = append(list, BrowserAccount{Name: name, Account: account})
	}
	return list
}

// CookiesFromBrowser returns the Cookie header for youtube.com from the first
// profile of that browser that holds a session. A browser with several profiles
// keeps one jar each, so every one is tried: the signed-in profile is not
// necessarily the first. A jar carrying the private session cookies
// (__Secure-3PSID and friends) wins over one that only signs requests.
func CookiesFromBrowser(name string) (string, error) {
	var partial string
	var lastErr error
	found := false
	for _, b := range Browsers() {
		if !strings.EqualFold(b.Name, name) {
			continue
		}
		found = true
		raw, err := jarCookies(b)
		if err != nil {
			lastErr = err
			continue
		}
		if sessionValue(raw) != "" {
			return raw, nil
		}
		if partial == "" && sidValue(raw) != "" {
			partial = raw
		}
	}
	if !found {
		return "", errors.New("no cookie jar for " + name)
	}
	if partial != "" {
		return partial, nil
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", errors.New(name + " is not signed in to YouTube")
}

// errKeyring reports a jar whose encrypted rows would not open with any password
// this machine offered. A locked keyring is the usual cause; a scheme Jota cannot
// read (Chrome's app-bound cookies) reads the same way, and unlocking is the one
// thing the user can try either way.
var errKeyring = errors.New("the browser's cookies could not be decrypted: unlock the keyring and try again")

// jarCookies turns one browser profile's jar into the Cookie header it would
// send to youtube.com, decrypting each value with the passwords that browser's
// keyring answered.
func jarCookies(b Browser) (string, error) {
	rows, version, err := readJar(b.Path)
	if err != nil {
		return "", err
	}
	secrets := chromiumSecrets(b.Name)
	header, locked := cookieHeader(rows, func(row jarRow) (string, error) {
		return decryptValue(row.encrypted, version, secrets)
	})
	if header == "" && locked {
		return "", errKeyring
	}
	return header, nil
}
