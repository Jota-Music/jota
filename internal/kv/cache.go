package kv

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// cached pairs a stored value with the fingerprint of the fetch that produced
// it, so a background refresh can tell whether the source still matches.
type cached[T any] struct {
	Value       T      `json:"value"`
	Fingerprint string `json:"fingerprint"`
}

// Cached returns the value stored at key, or runs fetch and stores the result.
// Entries never expire; Refreshable keeps them current in the background. When
// the store is unavailable it falls back to fetch so callers keep working
// without persistence.
func Cached[T any](b *Bucket, key string, fetch func() (T, error)) (T, error) {
	if err := EnsureStarted(); err != nil {
		return fetch()
	}

	var stored cached[T]
	if err := b.GetObject(key, &stored); err == nil && stored.Fingerprint != "" {
		return stored.Value, nil
	}

	value, err := fetch()
	if err != nil {
		var zero T
		return zero, err
	}

	_ = b.SetObject(key, cached[T]{Value: value, Fingerprint: Fingerprint(value)})
	return value, nil
}

// Refreshable refetches the value behind key and rewrites it only when it
// changed. It reports whether the stored entry was replaced; missing entries,
// fetch errors and unchanged content all report false.
func Refreshable[T any](b *Bucket, key string, fetch func() (T, error)) bool {
	if err := EnsureStarted(); err != nil {
		return false
	}

	var stored cached[T]
	if err := b.GetObject(key, &stored); err != nil || stored.Fingerprint == "" {
		return false
	}

	value, err := fetch()
	if err != nil {
		return false
	}

	fingerprint := Fingerprint(value)
	if fingerprint == stored.Fingerprint {
		return false
	}

	_ = b.SetObject(key, cached[T]{Value: value, Fingerprint: fingerprint})
	return true
}

// Fingerprint hashes a value into a compact change detector.
func Fingerprint(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
