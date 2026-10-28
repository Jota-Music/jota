package app

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// sessionPassKey is the settings key holding the shared passphrase used to
// encrypt sessions (YouTube cookies, Spotify credentials) before they travel
// through the sync document.
const sessionPassKey = "session-pass"

// sessionEncMagic marks a value encrypted with encryptWithPass.
const sessionEncMagic = "JOTAENC1:"

// SessionPass returns the shared passphrase.
func (a *App) SessionPass() string {
	return storedPass()
}

// storedPass reads the shared passphrase without an App, for the paths that
// apply a synced session at startup. The legacy youtube-cookie-pass key is
// still read as a fallback so an upgrade does not lose the pass.
func storedPass() string {
	var pass string
	if err := settings.GetObject(sessionPassKey, &pass); err == nil && pass != "" {
		return pass
	}
	_ = settings.GetObject("youtube-cookie-pass", &pass)
	return pass
}

// SetSessionPass stores the shared passphrase. When it is cleared, the staged
// sessions are dropped from the sync so they stop travelling. When it is set,
// what the document already staged is opened first: the pass is the only thing
// those blobs were waiting for, and staging the local session over them before
// applying them would throw the other device's session away.
func (a *App) SetSessionPass(pass string) error {
	if err := settings.SetObject(sessionPassKey, pass); err != nil {
		return err
	}
	if pass == "" {
		tombstoneSession("youtube")
		tombstoneSession("spotify")
		markSealed()
		a.syncInBackground()
		return nil
	}
	applyYouTubeCookies(stagedSession("youtube"))
	applySpotifySession(stagedSession("spotify"))
	markSealed()
	a.syncYouTubeCookies()
	a.syncSpotifySession()
	return nil
}

// encryptWithPass encrypts plain with pass (AES-GCM, key from PBKDF2). An empty
// pass leaves the value untouched, so callers can treat the result uniformly.
func encryptWithPass(pass, plain string) (string, error) {
	if pass == "" {
		return plain, nil
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := pbkdf2.Key([]byte(pass), salt, 100000, 32, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nil, nonce, []byte(plain), nil)
	out := append(append(salt, nonce...), ct...)
	return sessionEncMagic + base64.StdEncoding.EncodeToString(out), nil
}

// decryptWithPass reverses encryptWithPass. A value without the magic prefix is
// returned as-is (it was never encrypted).
func decryptWithPass(pass, data string) (string, error) {
	data = strings.TrimSpace(data)
	if !strings.HasPrefix(data, sessionEncMagic) {
		return data, nil
	}
	blob, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, sessionEncMagic))
	if err != nil {
		return "", err
	}
	if len(blob) < 16+12 {
		return "", fmt.Errorf("invalid encrypted payload")
	}
	salt := blob[:16]
	nonce := blob[16 : 16+12]
	ct := blob[16+12:]
	key := pbkdf2.Key([]byte(pass), salt, 100000, 32, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed (wrong pass?): %w", err)
	}
	return string(pt), nil
}
