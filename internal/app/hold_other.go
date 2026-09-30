//go:build !android

package app

// hold/release are Android-only: no other platform can kill the process out
// from under a browser redirect, so they are no-ops elsewhere.
func hold() {}

func release() {}
