//go:build !linux || !gtk3

package clipboard

// No asynchronous read here: request stays nil, Text reports ErrUnsupported, and
// the page falls back to the runtime's own clipboard, which does not block a GTK
// loop on these platforms.
