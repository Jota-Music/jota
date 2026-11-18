package app

import (
	"encoding/json"
	"strings"
	"sync/atomic"

	"github.com/Jota-Music/jota/internal/backup"
)

// sessionsSync is the sessions store wired into the backup document.
var sessionsSync = &sessionsStore{}

type sessionsStore struct {
	dirty atomic.Bool
}

func (s *sessionsStore) Section() (backup.Settings, error) {
	out := backup.Settings{}
	for _, key := range []string{"youtube", "spotify"} {
		entry, ok := stagedBlob(key)
		if !ok {
			continue
		}
		value, err := json.Marshal(entry.Value)
		if err != nil {
			continue
		}
		// An empty value is a tombstone: it travels so a session removed here
		// is also removed from the other devices instead of being restored.
		out[key] = backup.Setting{Value: value, At: entry.At}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (s *sessionsStore) Apply(part backup.Settings) error {
	for key, entry := range part {
		var blob string
		if err := json.Unmarshal(entry.Value, &blob); err != nil {
			continue
		}
		editSessions(func(sessions map[string]sessionEntry) {
			sessions[key] = sessionEntry{Value: blob, At: entry.At}
		})
		if blob == "" {
			continue
		}
		switch key {
		case "youtube":
			storeYouTubeSession(blob)
		case "spotify":
			applySpotifySession(blob)
		}
	}
	markSealed()
	return nil
}

// sealedSession reports whether a staged blob still needs a pass this device
// does not hold or cannot open it with.
func sealedSession(blob string) bool {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return false
	}
	pass := storedPass()
	if pass == "" {
		return true
	}
	_, err := decryptWithPass(pass, blob)
	return err != nil
}

// markSealed records in the status whether the document carries a session that
// is waiting for a pass, so the panel can say so instead of showing silence.
func markSealed() {
	sealed := sealedSession(stagedSession("youtube")) || sealedSession(stagedSession("spotify"))
	editSettings(func(current *deviceSettings) {
		status := current.Sync.Status
		if status.Sealed == sealed {
			return
		}
		status.Sealed = sealed
		current.Sync.Status = status
	})
}

// stagedSession reads the blob a pass staged for one source, empty when the
// source was tombstoned or never staged.
func stagedSession(key string) string {
	entry, _ := stagedBlob(key)
	return entry.Value
}

// tombstoneSession marks a session as removed with a write time, so the
// deletion travels to the other devices instead of the next merge restoring it.
func tombstoneSession(key string) {
	unstageBlob(key)
	sessionsSync.dirty.Store(true)
}

func (s *sessionsStore) Pending() bool {
	return s.dirty.Load()
}

func (s *sessionsStore) Synced() {
	s.dirty.Store(false)
}
