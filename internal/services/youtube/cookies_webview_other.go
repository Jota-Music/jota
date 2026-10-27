//go:build android || (!linux && !windows && !darwin)

package youtube

import "errors"

// WebviewCookies reads the in-app sign-in window's own cookie store, which only
// the desktop webviews keep. Android hands its session over through
// CookieManager instead, and it is the only one that can carry the HttpOnly
// cookies of a full session.
func WebviewCookies() (string, error) {
	return "", errors.New("reading the sign-in window's cookies is not supported on this platform")
}
