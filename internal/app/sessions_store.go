package app

import (
	"encoding/json"
	"log"
	"strings"
	"sync/atomic"
	"time"

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
		var blob string
		if err := settings.GetObject("session:"+key, &blob); err != nil {
			continue
		}
		value, err := json.Marshal(blob)
		if err != nil {
			continue
		}
		var at int64
		_ = settings.GetObject("session:"+key+settingAt, &at)
		// An empty value is a tombstone: it travels so a session removed here
		// is also removed from the other devices instead of being restored.
		out[key] = backup.Setting{Value: value, At: at}
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
		if err := settings.SetObject("session:"+key, blob); err != nil {
			return err
		}
		if err := settings.SetObject("session:"+key+settingAt, entry.At); err != nil {
			return err
		}
		if blob == "" {
			continue
		}
		switch key {
		case "youtube":
			applyYouTubeCookies(blob)
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
	var status SyncStatus
	if err := settings.GetObject(syncStatusKey, &status); err != nil {
		status = SyncStatus{}
	}
	if status.Sealed == sealed {
		return
	}
	status.Sealed = sealed
	if err := settings.SetObject(syncStatusKey, status); err != nil {
		log.Printf("sync status: %v", err)
	}
}

// stagedSession reads the blob a pass staged for one source, empty when the
// source was tombstoned or never staged.
func stagedSession(key string) string {
	var blob string
	_ = settings.GetObject("session:"+key, &blob)
	return blob
}

// tombstoneSession marks a session as removed with a write time, so the
// deletion travels to the other devices instead of the next merge restoring it.
func tombstoneSession(key string) {
	_ = settings.SetObject("session:"+key, "")
	_ = settings.SetObject("session:"+key+settingAt, time.Now().UnixMilli())
	sessionsSync.dirty.Store(true)
}

func (s *sessionsStore) Pending() bool {
	return s.dirty.Load()
}

func (s *sessionsStore) Synced() {
	s.dirty.Store(false)
}
