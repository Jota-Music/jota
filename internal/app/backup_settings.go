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

// carriedSettings are the settings a backup travels with: how to sync and the
// relay. Sessions travel in their own section.
var carriedSettings = []string{"sync", "relay"}

// settingsSync is the one settings store the app wires into its backup
// document, so a setting written through any caller marks the same store.
var settingsSync = &settingsStore{}

// settingsStore is the backup port over the settings bucket.
type settingsStore struct {
	dirty atomic.Bool
}

func (s *settingsStore) Section() (backup.Settings, error) {
	out := backup.Settings{}
	// sync
	v := readSyncSettings()
	syncVal := map[string]any{}
	if v.Import != nil {
		syncVal["import"] = *v.Import
	}
	if v.Interval != nil {
		syncVal["interval"] = *v.Interval
	}
	if v.Auto != nil {
		syncVal["auto"] = *v.Auto
	}
	if len(syncVal) > 0 {
		raw, err := json.Marshal(syncVal)
		if err == nil {
			var at int64
			_ = settings.GetObject("sync"+settingAt, &at)
			out["sync"] = backup.Setting{Value: raw, At: at}
		}
	}
	// relay
	var relay json.RawMessage
	if err := settings.GetObject("relay", &relay); err == nil {
		var at int64
		_ = settings.GetObject("relay"+settingAt, &at)
		out["relay"] = backup.Setting{Value: relay, At: at}
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

func touch(key string) {
	_ = settings.SetObject(key+settingAt, time.Now().UnixMilli())
	settingsSync.dirty.Store(true)
}

// Synced clears the pending mark after a pass wrote the settings out.
func Synced() {
	settingsSync.Synced()
}
