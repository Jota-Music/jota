package youtube

import (
	"errors"
	"os"
	"strings"
)

// errWebviewJar reports a sign-in window whose cookie store is not there yet.
// Every desktop webview keeps the session in memory and flushes it on its own
// schedule, so the caller polls instead of giving up on the first read.
var errWebviewJar = errors.New("the sign-in window has no cookie store yet")

// jarRow is one cookie as the browser stored it. value is the plaintext column,
// which Chromium leaves empty once it has encrypted the same cookie into
// encrypted_value.
type jarRow struct {
	host      string
	name      string
	value     string
	encrypted []byte
}

// cookieHeader turns jar rows into the Cookie header the jar's browser would
// send to youtube.com. The rows go back into the jar's own tab form so
// normalizeCookies keeps the youtube.com copy of each name over the google.com
// one, exactly as the browser would. A row that will not decrypt is skipped: a
// corrupt cookie must not cost the rest of the session. locked reports that
// something was skipped for that reason and nothing else came through.
func cookieHeader(rows []jarRow, decrypt func(jarRow) (string, error)) (string, bool) {
	lines := make([]string, 0, len(rows))
	locked := false
	for _, row := range rows {
		value := row.value
		if value == "" && len(row.encrypted) > 0 {
			var plain string
			var err error
			if decrypt != nil {
				plain, err = decrypt(row)
			} else {
				err = errors.New("encrypted cookie without a key")
			}
			if err != nil {
				locked = true
				continue
			}
			value = plain
		}
		if value == "" {
			continue
		}
		lines = append(lines, strings.Join([]string{row.host, "TRUE", "/", "TRUE", "0", row.name, value}, "\t"))
	}
	if len(lines) == 0 && locked {
		return "", true
	}
	return normalizeCookies(strings.Join(lines, "\n")), false
}

// cookieFile returns the first candidate that exists as a regular file.
func cookieFile(candidates ...string) string {
	for _, path := range candidates {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}
