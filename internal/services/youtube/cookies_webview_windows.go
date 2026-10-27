//go:build windows

package youtube

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

// WebviewCookies returns the Cookie header held by the in-app sign-in window's
// own store. document.cookie never carries the HttpOnly half of a Google
// session, so the store is the only way the window can hand over a real one.
func WebviewCookies() (string, error) {
	jar, state := webviewProfile()
	if jar == "" {
		return "", errWebviewJar
	}
	rows, version, err := readJar(jar)
	if err != nil {
		return "", err
	}
	key, keyErr := webviewKey(state)
	header, locked := cookieHeader(rows, func(row jarRow) (string, error) {
		if keyErr != nil {
			return "", keyErr
		}
		return decryptWindows(row.encrypted, version, key)
	})
	if header == "" && locked {
		return "", errors.New("the sign-in window's cookies could not be decrypted")
	}
	if header == "" {
		return "", errWebviewJar
	}
	return header, nil
}

// webviewProfile returns the cookie jar WebView2 keeps for the app and the Local
// State file that holds the key to it. The browser puts its user data folder
// beside the binary, and moves to the user's local app data when that copy is
// not writable, which is the case for an install under Program Files.
func webviewProfile() (string, string) {
	var roots []string
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Join(filepath.Dir(exe), filepath.Base(exe)+".WebView2"))
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		if exe, err := os.Executable(); err == nil {
			roots = append(roots, filepath.Join(local, filepath.Base(exe)+".WebView2"))
		}
	}
	for _, root := range roots {
		jar := cookieFile(filepath.Join(root, "EBWebView", "Default", "Network", "Cookies"))
		if jar != "" {
			return jar, filepath.Join(root, "EBWebView", "Local State")
		}
	}
	return "", ""
}

// webviewKey unwraps the AES key Chromium guards with the user's own logon
// session. The blob is the base64 of "DPAPI" followed by the protected key.
func webviewKey(state string) ([]byte, error) {
	raw, err := os.ReadFile(state)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		OSCrypt struct {
			EncryptedKey string `json:"encrypted_key"`
		} `json:"os_crypt"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	blob, err := base64.StdEncoding.DecodeString(parsed.OSCrypt.EncryptedKey)
	if err != nil {
		return nil, err
	}
	return dpapi(bytes.TrimPrefix(blob, []byte("DPAPI")))
}

func dpapi(blob []byte) ([]byte, error) {
	if len(blob) == 0 {
		return nil, errors.New("empty cookie key")
	}
	in := windows.DataBlob{Size: uint32(len(blob)), Data: &blob[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, 0, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return unsafe.Slice(out.Data, out.Size), nil
}

// decryptWindows turns one encrypted_value column into the cookie value.
// Chromium on Windows writes "v10", then AES-256-GCM with a nonce the scheme
// fixes at twelve bytes. Since jar version 24 the plaintext starts with the
// SHA-256 of the cookie's host, so the value cannot be replayed against another
// site. A "v20" row is app-bound encryption, which no key this user holds opens.
func decryptWindows(encrypted []byte, version int, key []byte) (string, error) {
	if len(encrypted) <= 3+12 || string(encrypted[:3]) != "v10" {
		return "", errors.New("unsupported cookie encryption")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, encrypted[3:3+12], encrypted[3+12:], nil)
	if err != nil {
		return "", fmt.Errorf("could not decrypt the cookie: %w", err)
	}
	if version >= 24 {
		if len(plain) <= sha256.Size {
			return "", errors.New("cookie carries no host prefix")
		}
		plain = plain[sha256.Size:]
	}
	return string(plain), nil
}
