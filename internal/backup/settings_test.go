package backup

import (
	"encoding/json"
	"testing"
)

// fakeSettings is the settings store a document carries, kept in memory so a
// test can assert what a merge wrote back.
type fakeSettings map[string]Setting

func (f fakeSettings) Section() (Settings, error) {
	if len(f) == 0 {
		return nil, nil
	}
	out := make(Settings, len(f))
	for key, entry := range f {
		out[key] = entry
	}
	return out, nil
}

func (f fakeSettings) Apply(part Settings) error {
	for key, entry := range part {
		f[key] = entry
	}
	return nil
}

func (f fakeSettings) Pending() bool {
	return false
}

func (f fakeSettings) Synced() {}

// dirtySettings is a store that remembers whether one of its settings changed
// since the last sync, the way the app's store does.
type dirtySettings struct {
	fakeSettings
	dirty bool
}

func (d *dirtySettings) Pending() bool {
	return d.dirty
}

func (d *dirtySettings) Apply(part Settings) error {
	if err := d.fakeSettings.Apply(part); err != nil {
		return err
	}
	d.dirty = true
	return nil
}

func (d *dirtySettings) Synced() {
	d.dirty = false
}

// A setting travels with the moment it was written: the newest copy wins, an
// explicit restore lets the document win, and a pass that does not apply answers
// the union while the store keeps its own value.
func TestMergeSettings(t *testing.T) {
	local := Setting{Value: json.RawMessage(`"30m"`), At: 20}
	cases := []struct {
		name    string
		remote  Setting
		restore bool
		apply   bool
		union   string
		stored  string
		changed int
	}{
		{"the newest copy wins", Setting{Value: json.RawMessage(`"5m"`), At: 10}, false, true, `"30m"`, `"30m"`, 0},
		{"a newer document wins", Setting{Value: json.RawMessage(`"5m"`), At: 30}, false, true, `"5m"`, `"5m"`, 1},
		{"a restore lets the document win", Setting{Value: json.RawMessage(`"5m"`), At: 10}, true, true, `"5m"`, `"5m"`, 1},
		{"without apply the store keeps its own", Setting{Value: json.RawMessage(`"5m"`), At: 30}, false, false, `"5m"`, `"30m"`, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			store := &dirtySettings{fakeSettings: fakeSettings{"sync-interval": local}}
			out, changed, err := mergeSettings(store, Settings{"sync-interval": c.remote}, c.restore, c.apply)
			if err != nil {
				t.Fatalf("merge: %v", err)
			}
			if changed != c.changed {
				t.Fatalf("changed = %d, want %d", changed, c.changed)
			}
			if got := string(out["sync-interval"].Value); got != c.union {
				t.Fatalf("union = %s, want %s", got, c.union)
			}
			if got := string(store.fakeSettings["sync-interval"].Value); got != c.stored {
				t.Fatalf("store = %s, want %s", got, c.stored)
			}
		})
	}
}

// The manual sync is offered while a carried setting changed, and a pass that
// applied the settings answers it. A pass that keeps the store's own values
// writes nothing, so the change is still waiting.
func TestSettingsPending(t *testing.T) {
	entry := Setting{Value: json.RawMessage(`"30m"`), At: 20}
	store := &dirtySettings{fakeSettings: fakeSettings{"sync-interval": entry}}
	remote := Settings{"sync-interval": entry}
	service := &Service{Settings: store}

	store.dirty = true
	if !service.Pending() {
		t.Fatal("a changed setting was not reported as pending")
	}

	if _, _, err := mergeSettings(store, remote, false, true); err != nil {
		t.Fatalf("merge: %v", err)
	}
	if service.Pending() {
		t.Fatal("a pass that applied the settings left them pending")
	}

	store.dirty = true
	if _, _, err := mergeSettings(store, remote, false, false); err != nil {
		t.Fatalf("merge: %v", err)
	}
	if !service.Pending() {
		t.Fatal("a pass that did not apply the settings cleared them")
	}
}
