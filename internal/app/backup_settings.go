package app

import (
	"encoding/json"
	"slices"
	"sync/atomic"
	"time"

	"github.com/Jota-Music/jota/internal/backup"
)

// settingAt is the companion key that remembers when a carried setting was
// written.
const settingAt = "-at"

// carriedSettings are the settings a backup travels with: when and how to sync,
// and the relay. A folder path belongs to the device, the sync status is this
// device's own state, and the rest is either machine-specific or something the
// user did not ask to copy.
var carriedSettings = []string{"sync-interval", "sync-import", "relay"}

// settingsSync is the one settings store the app wires into its backup
// document, so a setting written through any caller marks the same store.
var settingsSync = &settingsStore{}

// settingsStore is the backup port over the settings bucket.
type settingsStore struct {
	dirty atomic.Bool
}

func (s *settingsStore) Section() (backup.Settings, error) {
	out := backup.Settings{}
	for _, key := range carriedSettings {
		var value json.RawMessage
		if err := settings.GetObject(key, &value); err != nil {
			continue
		}
		var at int64
		_ = settings.GetObject(key+settingAt, &at)
		out[key] = backup.Setting{Value: value, At: at}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (s *settingsStore) Apply(part backup.Settings) error {
	applied := false
	for key, entry := range part {
		if len(entry.Value) == 0 || !slices.Contains(carriedSettings, key) {
			continue
		}
		if err := settings.SetObject(key, entry.Value); err != nil {
			return err
		}
		if err := settings.SetObject(key+settingAt, entry.At); err != nil {
			return err
		}
		applied = true
	}
	if applied {
		s.dirty.Store(true)
	}
	return nil
}

func (s *settingsStore) Pending() bool {
	return s.dirty.Load()
}

// Synced clears the pending mark after a pass wrote the settings out.
func (s *settingsStore) Synced() {
	s.dirty.Store(false)
}

// touch records when a carried setting was written, so the newest copy wins a
// merge instead of an older document overwriting it.
func touch(key string) {
	_ = settings.SetObject(key+settingAt, time.Now().UnixMilli())
	settingsSync.dirty.Store(true)
}

// Synced clears the pending mark after a pass wrote the settings out.
func Synced() {
	settingsSync.Synced()
}
