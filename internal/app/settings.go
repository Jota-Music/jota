package app

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/state"
)

// deviceSettings is what the device keeps of the app's own settings: the sync
// folder and its state, the relay the app dials, the pass the sessions are
// sealed with and the switches the UI owns.
type deviceSettings struct {
	Sync  syncOptions `json:"sync,omitempty"`
	Relay Relay       `json:"relay,omitempty"`
	// RelayAt is when the relay was written, so a pass can tell which copy is
	// the newer one.
	RelayAt     int64  `json:"relayAt,omitempty"`
	SessionPass string `json:"sessionPass,omitempty"`
	// IdleReload is a pointer because it was on before it could be turned off:
	// a device that never touched it has to keep the behaviour it had.
	IdleReload *bool `json:"idleReload,omitempty"`
	Discord    bool  `json:"discord,omitempty"`
}

// syncOptions is the sync part: where the document lives, whether a pass
// applies it, how often it runs and what the last pass did. The switches are
// pointers because the document that carries them may leave one out, and a
// missing switch means the behaviour that existed before it did.
type syncOptions struct {
	Folder   string     `json:"folder,omitempty"`
	Import   *bool      `json:"import,omitempty"`
	Auto     *bool      `json:"auto,omitempty"`
	Interval *int       `json:"interval,omitempty"`
	Status   SyncStatus `json:"status,omitempty"`
	// At is when the carried settings were written, so a pass can tell which
	// copy is the newer one.
	At int64 `json:"at,omitempty"`
}

// sessionEntry is one session waiting to travel in the document: the sealed
// blob and the moment it was staged.
type sessionEntry struct {
	Value string `json:"value,omitempty"`
	At    int64  `json:"at,omitempty"`
}

// settingsMu serializes the read-modify-write of the settings section. A write
// replaces the whole section, so two of them interleaved — a pass recording its
// status while the user flips a switch — would drop one of the two.
var settingsMu sync.Mutex

// editSettings changes the settings under the lock that makes the whole
// read-modify-write one step.
func editSettings(change func(*deviceSettings)) {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	current := readSettings()
	change(&current)
	writeSettings(current)
}

// sessionsMu is the same lock for the sessions section, which is written whole
// for the same reason.
var sessionsMu sync.Mutex

// editSessions changes the staged sessions under their lock.
func editSessions(change func(map[string]sessionEntry)) {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	sessions := readSessions()
	if sessions == nil {
		sessions = map[string]sessionEntry{}
	}
	change(sessions)
	writeSessions(sessions)
}

// readSettings reads the settings the device keeps.
func readSettings() deviceSettings {
	out, _ := state.Read[deviceSettings](state.Settings)
	return out
}

// writeSettings stores the settings the device keeps.
func writeSettings(s deviceSettings) {
	if err := state.Write(state.Settings, s); err != nil {
		log.Printf("settings: %v", err)
	}
}

// readSessions reads the sessions the device stages for the document.
func readSessions() map[string]sessionEntry {
	out, _ := state.Read[map[string]sessionEntry](state.Sessions)
	return out
}

// writeSessions stores the sessions the device stages for the document.
func writeSessions(sessions map[string]sessionEntry) {
	if err := state.Write(state.Sessions, sessions); err != nil {
		log.Printf("sessions: %v", err)
	}
}

// stagedBlob reads one staged session.
func stagedBlob(source string) (sessionEntry, bool) {
	entry, ok := readSessions()[source]
	return entry, ok
}

// stageBlob stores one staged session and dates it.
func stageBlob(source string, blob string) error {
	editSessions(func(sessions map[string]sessionEntry) {
		sessions[source] = sessionEntry{Value: blob, At: time.Now().UnixMilli()}
	})
	return nil
}

// unstageBlob drops one staged session, leaving the date behind: an empty
// value is a tombstone that travels, so a session removed here is also removed
// from the other devices.
func unstageBlob(source string) {
	editSessions(func(sessions map[string]sessionEntry) {
		sessions[source] = sessionEntry{At: time.Now().UnixMilli()}
	})
}

// syncValue is the compound value the document carries for the sync settings.
func syncValue(options syncOptions) (json.RawMessage, bool) {
	value := map[string]any{}
	if options.Import != nil {
		value["import"] = *options.Import
	}
	if options.Interval != nil {
		value["interval"] = *options.Interval
	}
	if options.Auto != nil {
		value["auto"] = *options.Auto
	}
	if len(value) == 0 {
		return nil, false
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	return raw, true
}

// syncValueOf reads the compound value back.
func syncValueOf(raw json.RawMessage) syncOptions {
	var options syncOptions
	if err := json.Unmarshal(raw, &options); err != nil {
		return syncOptions{}
	}
	return options
}
