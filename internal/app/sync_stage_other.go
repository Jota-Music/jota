//go:build !android

package app

// syncStageDir has no directory off Android: a folder is a filesystem path
// there, so the document is read and written directly instead of being staged.
func syncStageDir() string {
	return ""
}
