package kv

import "time"

// Cached returns the value stored at key, or runs fetch, stores the result
// with ttl and returns it. When the store is unavailable it falls back to
// fetch so callers keep working without persistence.
func Cached[T any](b *Bucket, key string, ttl time.Duration, fetch func() (T, error)) (T, error) {
	if err := EnsureStarted(); err != nil {
		return fetch()
	}

	var cached T
	if err := b.GetObject(key, &cached); err == nil {
		return cached, nil
	}

	value, err := fetch()
	if err != nil {
		var zero T
		return zero, err
	}

	_ = b.SetObject(key, value, ttl)
	return value, nil
}
