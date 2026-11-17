//go:build linux

package youtube

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// keyringTimeout bounds one keyring answer. A locked wallet opens a prompt and
// the command waits on it, and this runs on a request the settings panel is
// waiting for, so a keyring that never answers must not hold it.
const keyringTimeout = 5 * time.Second

// keyringCommand runs a keyring tool with the deadline above.
func keyringCommand(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), keyringTimeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

var (
	keyringMu      sync.Mutex
	keyringSecrets = map[string][]string{}
	keyringed      = map[string]bool{}
)

// keyringSecret returns the passwords the browser's keyring items hold, best
// first, or nothing when this desktop keeps none. The answer is cached for the
// run: a jar read asks once per encrypted cookie, and every miss would otherwise
// spawn a process per cookie.
//
// These are secrets. They are never logged and never leave this package.
func keyringSecret(browser string) []string {
	keyringMu.Lock()
	defer keyringMu.Unlock()
	if keyringed[browser] {
		return keyringSecrets[browser]
	}
	keyringed[browser] = true
	secrets := libsecret(browser)
	if len(secrets) == 0 {
		secrets = kwallet(browser)
	}
	keyringSecrets[browser] = secrets
	return secrets
}

// libsecret asks the GNOME keyring, which is where the Chromium family stores
// "<Browser> Safe Storage" on anything that is not KDE.
func libsecret(browser string) []string {
	out, err := keyringCommand("secret-tool", "search", "--all",
		"xdg:schema", "chrome_libsecret_os_crypt_password_v2")
	if err != nil {
		return nil
	}

	label := keyringLabel(browser)
	want := strings.ToLower(browser)
	var own, other []string
	for _, block := range strings.Split(string(out), "\n[") {
		fields := map[string]string{}
		for line := range strings.SplitSeq(block, "\n") {
			if key, value, ok := strings.Cut(strings.TrimSpace(line), " = "); ok {
				fields[key] = value
			}
		}
		if !strings.EqualFold(fields["label"], label) || fields["secret"] == "" {
			continue
		}
		// Other apps built on Chromium register the same label, so the browser
		// itself is the one that names itself in attribute.application. Try that
		// first and keep the rest as a fallback.
		if strings.Contains(strings.ToLower(fields["attribute.application"]), want) {
			own = append(own, fields["secret"])
		} else {
			other = append(other, fields["secret"])
		}
	}
	return append(own, other...)
}

// kwallet asks KDE's wallet, which keeps the same password under the "<Browser>
// Keys" folder.
func kwallet(browser string) []string {
	product := strings.TrimSuffix(keyringLabel(browser), " Safe Storage")
	for _, entry := range []string{keyringLabel(browser), "Chrome Safe Storage"} {
		out, err := keyringCommand("kwallet-query", "-f", product+" Keys", "-r", entry, "kwallet")
		if err != nil {
			continue
		}
		if secret := strings.TrimSpace(string(out)); secret != "" {
			return []string{secret}
		}
	}
	return nil
}
