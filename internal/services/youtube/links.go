package youtube

import (
	"os"
	"path/filepath"

	"github.com/Jota-Music/jota/internal/cache"
)

// pickName is where the video an earlier search settled on is kept: one file
// per song, with no extension, so a play rewrites one entry and not the whole
// state. The content is the 11 bytes of the video id, read directly.
func pickName(song string) string { return "youtube/links/" + song }

// pinned returns the video the user fixed for a song by hand: an order, kept
// in the state and never replaced by a resolve.
func pinned(song string) string {
	return readSession().Links[song]
}

// pin records the user's pick, or drops it when video is empty. The caller
// validates the id.
func pin(song, video string) error {
	return updateSession(func(stored *session) {
		if stored.Links == nil {
			stored.Links = map[string]string{}
		}

		// Invalidate the cached stream so the new video is resolved on the next
		// request: the old one still plays, but it is not what the user picked.
		// Only the resolve is dropped, not the relay: if that video happens to be
		// the one playing, its ranges have to keep being served.
		if old := stored.Links[song]; old != "" && old != video {
			forgetStream(old)
		}

		if video == "" {
			delete(stored.Links, song)
			// Unpinning means forget my choice, so the old pick goes too:
			// otherwise it resurfaces and the pin looks like it did nothing.
			forgetPick(song)
		} else {
			stored.Links[song] = video
		}
	})
}

// forgetPin drops the user's pin without touching the pick a search settled on:
// a pin that points at a dead video is useless, but the pick that replaced it
// is still worth keeping.
func forgetPin(song string) error {
	return updateSession(func(stored *session) {
		if stored.Links != nil {
			delete(stored.Links, song)
		}
	})
}

// forgetPick drops the pick an earlier search settled on.
func forgetPick(song string) {
	if p, err := cache.RawPath(pickName(song)); err == nil {
		_ = os.Remove(p)
	}
}

// remembered returns the video an earlier search settled on.
func remembered(song string) string {
	p, err := cache.RawPath(pickName(song))
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	// A file that does not hold a video id is damage, never a pick.
	id := string(data)
	if !isYoutubeId(id) {
		return ""
	}
	return id
}

// remember stores the pick a resolve settled on. Losing it costs one search.
func remember(song, video string) {
	if song == "" || video == "" {
		return
	}
	p, err := cache.RawPath(pickName(song))
	if err != nil {
		return
	}
	// A play that resolved the same video again has nothing to write.
	if current, err := os.ReadFile(p); err == nil && string(current) == video {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(video), 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, p)
}

// MigrateLinks moves the picks the old layout kept in the state into the cache,
// so the state only holds pins. The flag is what makes it once per device: the
// map it reads is the same one the pins live in now, so a second run would turn
// the user's pins into picks a resolve may overwrite.
var MigrateLinks = func() {
	if readSession().LinksMigrated {
		return
	}
	_ = updateSession(func(stored *session) {
		if stored.LinksMigrated {
			return
		}
		for song, video := range stored.Links {
			if video != "" {
				remember(song, video)
			}
		}
		stored.Links = nil
		stored.LinksMigrated = true
	})
}
