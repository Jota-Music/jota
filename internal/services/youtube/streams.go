package youtube

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/cache"
	"github.com/Jota-Music/jota/internal/music"
)

// streamName is where the stream resolved for a video is kept: one file per
// video, so a play rewrites one entry and not the whole YouTube cache.
func streamName(videoID string) string { return "youtube/streams/" + videoID }

// streamMaxAge is when a stream file is assumed dead whatever it holds. How
// long an entry is trusted comes from the URL's own expiry, which is about six
// hours; this is the ceiling for the ones nobody reads again, so the sweep
// never deletes a stream that could still play.
const streamMaxAge = 24 * time.Hour

// freshFor is how long a stream this run resolved is played without probing its
// URL: the resolve already probed it, and the signature has hours left.
const freshFor = 30 * time.Minute

// probeGrace is how recently a stream must have been resolved for its URL to be
// trusted without a HEAD: audioURL probed it then, and an early revocation that
// close to the resolve is unlikely. Older entries are still probed.
const probeGrace = 10 * time.Minute

// fresh holds the streams resolved by this run. A HEAD to googlevideo costs a
// round trip, and paying it again for a URL this process just proved playable
// only slows down the next track.
var fresh sync.Map // videoID -> time.Time

// cachedAudio returns the stream resolved for a video, when it can still be
// played: not expired, and the URL still answering.
func cachedAudio(videoID string) *music.Audio {
	audio, saved, ok := cache.GetWithMeta[music.Audio](streamName(videoID))
	if !ok {
		return nil
	}
	// A stream resolved before the preferred container changed would keep
	// playing the one this player cannot seek, so it is resolved again.
	if preferWebM != strings.HasPrefix(streamMime(audio.Url), "audio/webm") {
		return nil
	}
	switch {
	case time.Since(time.UnixMilli(saved)) < probeGrace:
		// Resolved a few minutes ago: audioURL already probed the URL and the
		// signature has hours left. No need to pay a HEAD again.
	case isFresh(videoID):
		// An entry this run resolved was probed minutes ago and its own expiry,
		// which is what the cache drops the entry by, is still hours away.
	case !audioCacheStillValid(&audio):
		return nil
	}
	audio.Url = ProxyURL(videoID, audio.Url)
	return &audio
}

// isFresh reports whether this run resolved the stream a few minutes ago.
func isFresh(videoID string) bool {
	at, ok := fresh.Load(videoID)
	return ok && time.Since(at.(time.Time)) < freshFor
}

// keepAudio stores the stream resolved for a video, for as long as its URL lives.
func keepAudio(videoID string, audio music.Audio, ttl time.Duration) {
	_ = cache.Put(streamName(videoID), audio, ttl)
	fresh.Store(videoID, time.Now())
}

// forgetAudio drops the stream of a video: the user picked another one.
func forgetAudio(videoID string) {
	fresh.Delete(videoID)
	_ = cache.Forget(streamName(videoID))
	proxyMu.Lock()
	delete(proxyStreams, videoID)
	proxyMu.Unlock()
}

// PurgeStreams drops the streams whose URL died with its signature. The age of
// the file is enough to tell: no googlevideo URL lives longer than a day.
func PurgeStreams() {
	if dropped := cache.Sweep("youtube/streams", streamMaxAge); dropped > 0 {
		log.Printf("youtube: %d expired streams dropped", dropped)
	}
}
