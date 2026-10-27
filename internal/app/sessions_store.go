package app

import (
	"encoding/json"
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
	return nil
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
