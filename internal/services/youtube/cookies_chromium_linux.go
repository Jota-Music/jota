//go:build linux

package youtube

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// chromiumIV is the fixed IV every Linux Chromium build uses: sixteen spaces,
// not a random one, which is why the scheme is only as strong as the password.
var chromiumIV = bytes.Repeat([]byte(" "), aes.BlockSize)

// chromiumSecrets lists the passwords a Linux Chromium build may have used, in
// the order worth trying: whatever the keyring answered, then the constants used
// when no keyring kept one. yt-dlp tries the same set.
func chromiumSecrets(browser string) []string {
	return slices.Concat(keyringSecret(browser), noKeyringSecrets)
}

// noKeyringSecrets are the passwords Chromium falls back to on a Linux desktop
// that has no keyring: a literal, and then nothing at all.
var noKeyringSecrets = []string{"peanuts", ""}

// decryptValue turns one encrypted_value column into the cookie value. Chromium
// writes "v10"/"v11", then AES-128-CBC with a key derived from the browser's
// keyring password. Since jar version 24 the plaintext starts with the SHA-256
// of the cookie's host, so the value cannot be replayed against another site.
func decryptValue(encrypted []byte, version int, secrets []string) (string, error) {
	if len(encrypted) <= 3 {
		return "", errors.New("encrypted cookie too short")
	}
	scheme := string(encrypted[:3])
	if scheme != "v10" && scheme != "v11" {
		return "", fmt.Errorf("unsupported cookie encryption %q", scheme)
	}

	body := encrypted[3:]
	for _, secret := range secrets {
		plain, err := decryptCBC(secret, body)
		if err != nil {
			continue
		}
		if version >= 24 {
			if len(plain) <= sha256.Size {
				continue
			}
			plain = plain[sha256.Size:]
		}
		return string(plain), nil
	}
	return "", errors.New("could not decrypt the cookie")
}

// decryptCBC unwraps one encrypted_value body with the key derived from secret.
func decryptCBC(secret string, body []byte) ([]byte, error) {
	if len(body) == 0 || len(body)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a whole block")
	}
	block, err := aes.NewCipher(derive(secret))
	if err != nil {
		return nil, err
	}
	plain := make([]byte, len(body))
	cipher.NewCBCDecrypter(block, chromiumIV).CryptBlocks(plain, body)
	return unpad(plain)
}

// derive is Chromium's key stretching: PBKDF2-SHA1 with the literal "saltysalt"
// and a single iteration, which is a fixed table anyone can precompute — the
// keyring password is the only secret left.
func derive(secret string) []byte {
	return pbkdf2.Key([]byte(secret), []byte("saltysalt"), 1, 16, sha1.New)
}

// unpad strips PKCS#7 padding. Valid padding is also how a wrong key is told
// apart from a cookie that actually decrypted.
func unpad(plain []byte) ([]byte, error) {
	n := int(plain[len(plain)-1])
	if n == 0 || n > aes.BlockSize || n > len(plain) {
		return nil, errors.New("bad padding")
	}
	if !bytes.Equal(plain[len(plain)-n:], bytes.Repeat([]byte{byte(n)}, n)) {
		return nil, errors.New("bad padding")
	}
	return plain[:len(plain)-n], nil
}

// keyringLabel is the name the browser gives its keyring item. Most use their own
// product name, Edge signs its items "Microsoft Edge".
func keyringLabel(browser string) string {
	if strings.EqualFold(browser, "edge") {
		return "Microsoft Edge Safe Storage"
	}
	return browser + " Safe Storage"
}
