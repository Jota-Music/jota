//go:build linux

package youtube

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"testing"
)

// seal builds one encrypted_value the way a Linux Chromium would write it.
func seal(t *testing.T, scheme, secret string, plain []byte) []byte {
	t.Helper()
	block, err := aes.NewCipher(derive(secret))
	if err != nil {
		t.Fatal(err)
	}
	n := aes.BlockSize - len(plain)%aes.BlockSize
	padded := append(bytes.Clone(plain), bytes.Repeat([]byte{byte(n)}, n)...)
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, chromiumIV).CryptBlocks(out, padded)
	return append([]byte(scheme), out...)
}

func TestDecryptValue(t *testing.T) {
	host := sha256.Sum256([]byte(".youtube.com"))
	bound := append(host[:], []byte("secret-value")...)

	cases := []struct {
		name    string
		blob    []byte
		version int
		secrets []string
		want    string
	}{
		{
			name:    "v10 keyring secret",
			blob:    seal(t, "v10", "keyring-secret", []byte("value")),
			version: 19,
			secrets: []string{"keyring-secret", "peanuts", ""},
			want:    "value",
		},
		{
			name:    "v10 falls back to peanuts",
			blob:    seal(t, "v11", "peanuts", []byte("value")),
			version: 19,
			secrets: []string{"peanuts", ""},
			want:    "value",
		},
		{
			name:    "v24 drops the host prefix",
			blob:    seal(t, "v10", "keyring-secret", bound),
			version: 27,
			secrets: []string{"keyring-secret"},
			want:    "secret-value",
		},
		{
			name:    "version 19 keeps the prefix",
			blob:    seal(t, "v10", "keyring-secret", bound),
			version: 19,
			secrets: []string{"keyring-secret"},
			want:    string(bound),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := decryptValue(c.blob, c.version, c.secrets)
			if err != nil {
				t.Fatalf("decryptValue: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestDecryptValueRejects(t *testing.T) {
	cases := []struct {
		name string
		blob []byte
	}{
		{"empty", nil},
		{"truncated scheme", []byte("v1")},
		{"app-bound scheme", append([]byte("v20"), seal(t, "v10", "peanuts", []byte("x"))[3:]...)},
		{"not a whole block", append([]byte("v10"), 1, 2, 3)},
		{"wrong secret", seal(t, "v10", "another-secret", []byte("value"))},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, err := decryptValue(c.blob, 19, []string{"peanuts", ""}); err == nil {
				t.Fatalf("decrypted %q, want an error", got)
			}
		})
	}
}
