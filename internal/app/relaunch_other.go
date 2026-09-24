//go:build !android

package app

// relaunchAfterUpdate reports whether the caller should quit after a
// successful install so a scheduled relaunch can pick up the new binary. On
// Android the system package installer takes over, so this is false.
func relaunchAfterUpdate() bool { return true }
