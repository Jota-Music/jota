package app

import (
	"encoding/json"
	"slices"
	"sync/atomic"
	"time"

	"github.com/Jota-Music/jota/internal/backup"
)

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
	current := readSettings()
	// sync
	if raw, ok := syncValue(current.Sync); ok {
		out["sync"] = backup.Setting{Value: raw, At: current.Sync.At}
	}
	// relay
	if raw, err := json.Marshal(current.Relay); err == nil && (current.Relay.URL != "" || current.Relay.Token != "") {
		out["relay"] = backup.Setting{Value: raw, At: current.RelayAt}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (s *settingsStore) Apply(part backup.Settings) error {
	applied := false
	editSettings(func(current *deviceSettings) {
		for key, entry := range part {
			if len(entry.Value) == 0 || !slices.Contains(carriedSettings, key) {
				continue
			}
			switch key {
			case "sync":
				options := syncValueOf(entry.Value)
				current.Sync.Import = options.Import
				current.Sync.Auto = options.Auto
				current.Sync.Interval = options.Interval
				current.Sync.At = entry.At
			case "relay":
				var relay Relay
				if err := json.Unmarshal(entry.Value, &relay); err != nil {
					continue
				}
				current.Relay = relay
				current.RelayAt = entry.At
			}
			applied = true
		}
	})
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

// touchSync dates the carried sync settings.
func touchSync() {
	editSettings(func(current *deviceSettings) {
		current.Sync.At = time.Now().UnixMilli()
	})
	settingsSync.dirty.Store(true)
}

// touchRelay dates the carried relay settings.
func touchRelay() {
	editSettings(func(current *deviceSettings) {
		current.RelayAt = time.Now().UnixMilli()
	})
	settingsSync.dirty.Store(true)
}

// Synced clears the pending mark after a pass wrote the settings out.
func Synced() {
	settingsSync.Synced()
}
