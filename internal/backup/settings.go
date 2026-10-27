package backup

import (
	"bytes"
	"encoding/json"
)

// Setting is one carried value: the raw value its owner stored, and the moment
// it was written, so the newest copy wins a merge.
type Setting struct {
	Value json.RawMessage `json:"value"`
	At    int64           `json:"at"`
}

// Settings is the settings part of a backup document, keyed by the setting's
// name in the settings bucket. Only the settings that make sense on another
// device travel here; a local path or the state of this device stays out.
type Settings map[string]Setting

// Sessions is the sessions part of a backup document, keyed by source (youtube,
// spotify). Each value is the encrypted blob its source stored, carried exactly
// like a setting.
type Sessions = Settings

// SettingsStore is a section the document carries as settings. The app
// implements it over its own bucket, so the document never touches storage
// itself; the sessions section uses the same port.
type SettingsStore interface {
	Section() (Settings, error)
	Apply(Settings) error
	// Pending reports whether a carried setting changed since the last sync.
	Pending() bool
	// Synced clears the pending mark once a pass wrote the settings out.
	Synced()
}

// mergeSettings reconciles the settings section with the store and answers the
// section both sides agree on, plus how many keys changed. The newest write of
// each key wins; an explicit restore lets the document win outright. With apply
// off the union is answered but the store keeps its own values, so a change made
// here is still waiting to travel.
func mergeSettings(store SettingsStore, remote Settings, restore, apply bool) (Settings, int, error) {
	local, err := store.Section()
	if err != nil {
		return Settings{}, 0, err
	}
	out := union(remote, local, restore)
	if !apply {
		return out, 0, nil
	}
	changed := diff(local, out)
	if changed > 0 {
		if err := store.Apply(out); err != nil {
			return out, 0, err
		}
	}
	// The document now carries every key, whichever side it came from.
	store.Synced()
	return out, changed, nil
}

// union keeps every key of both sides, taking the copy that was written last
// unless restore says the document wins.
func union(remote, local Settings, restore bool) Settings {
	out := make(Settings, len(local)+len(remote))
	for key, entry := range local {
		out[key] = entry
	}
	for key, entry := range remote {
		if cur, ok := out[key]; ok && !restore && cur.At >= entry.At {
			continue
		}
		out[key] = entry
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// diff counts the keys whose value or age the union moves.
func diff(local, out Settings) int {
	changed := 0
	for key, entry := range out {
		cur, ok := local[key]
		if !ok || cur.At != entry.At || !bytes.Equal(cur.Value, entry.Value) {
			changed++
		}
	}
	return changed
}
