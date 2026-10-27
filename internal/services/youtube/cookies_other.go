//go:build !linux || android

package youtube

import "errors"

// Reading a browser's cookie jar is Linux-only for now. Android has no in-app
// sign-in and the desktop ports have no jar reader yet, so the UI must report the
// feature as unavailable rather than offer a button that cannot work.
func CookiesBrowsers() []string {
	return nil
}

// BrowserAccount is one browser and the account its jar is signed in to.
type BrowserAccount struct {
	Name    string
	Account string
}

func BrowserAccounts() []BrowserAccount {
	return nil
}

func CookiesFromBrowser(string) (string, error) {
	return "", errors.New("importing cookies from a browser is not supported on this platform")
}
