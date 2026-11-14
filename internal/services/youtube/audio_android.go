//go:build android

package youtube

// preferWebM picks the Opus/WebM stream over the AAC/MP4 one. The Android
// WebView hands src playback to the platform player, which seeks a fragmented
// MP4 by an estimated byte offset: the offset lands inside a fragment, the
// player reads to the end of the file and reports the track as finished. WebM
// carries a cue index, so the same seek asks for the exact cluster it needs.
var preferWebM = true
