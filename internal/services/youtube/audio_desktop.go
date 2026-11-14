//go:build !android

package youtube

// preferWebM is off where the player seeks the MP4 stream correctly: WebKitGTK
// and macOS start faster on AAC and can stall on googlevideo's WebM/Opus.
var preferWebM = false
