// Package state owns what the app keeps between launches: the device's state
// and the cache of what the sources answer.
//
// The state is one document (state.json) with a named section per feature, and
// one writer puts it on disk. The cache is a file per entity, under a name that
// says what it is.
package state

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// The sections of the document.
const (
	Playlists = "playlists"
	Follows   = "follows"
	Saved     = "saved"
	Order     = "order"
	Rooms     = "rooms"
	Settings  = "settings"
	Sessions  = "sessions"
	Spotify   = "spotify"
	YouTube   = "youtube"
)

// debounce is how long a burst of changes waits before the document is written.
const debounce = 150 * time.Millisecond

// Store is the document in memory.
type Store struct {
	mu       sync.Mutex
	where    string
	sections map[string]json.RawMessage
	dirty    bool
	wake     chan struct{}
}

var (
	current    = &Store{sections: map[string]json.RawMessage{}, wake: make(chan struct{}, 1)}
	writerOnce sync.Once
)

// Current is the device's state.
func Current() *Store { return current }

func (s *Store) path() (string, error) {
	base, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "jota", "state.json"), nil
}

func (s *Store) load() error {
	path, err := s.path()
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.where == path && s.sections != nil {
		return nil
	}
	s.where = path
	s.sections = map[string]json.RawMessage{}
	s.dirty = false

	raw, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("state: cannot read the document: %v", err)
		}
		return nil
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		// The document is kept aside so the user can recover it by hand: it is
		// the only copy of what the app knew.
		log.Printf("state: the document will not decode, it is kept as %s", path+".bad")
		_ = os.Rename(path, path+".bad")
		return nil
	}
	s.sections = top
	return nil
}

func (s *Store) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirty {
		return nil
	}
	encoded, err := json.Marshal(s.sections)
	if err != nil {
		return err
	}
	path, err := s.path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := writeSync(tmp, encoded); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	// The rename is only durable once the directory itself is flushed: without
	// this the name can be lost while the file it pointed at stays.
	_ = syncDir(filepath.Dir(path))
	s.dirty = false
	return nil
}

// writeSync writes data to a file and flushes it to the disk, so a crash
// leaves the whole file and not a half of one.
func writeSync(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// syncDir flushes a directory entry. A platform that does not allow it (a
// Windows directory, for one) answers with an error that costs nothing: the
// file itself is already on the disk.
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func (s *Store) markDirty() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Start keeps the document on disk via a writer thread.
func Start() {
	writerOnce.Do(func() {
		// A write that died between the temporary file and the rename left it
		// behind, and nothing else would ever collect it.
		if path, err := current.path(); err == nil {
			_ = os.Remove(path + ".tmp")
		}
		go func() {
			for range current.wake {
				time.Sleep(debounce)
				_ = current.save()
			}
		}()
	})
}

// Reset forgets the document held in memory.
func Reset() {
	current.mu.Lock()
	current.where = ""
	current.sections = map[string]json.RawMessage{}
	current.dirty = false
	current.mu.Unlock()
}

// Close writes the document that is still pending.
func Close() {
	_ = current.save()
}

// Read returns a section, decoded.
func Read[T any](name string) (T, error) {
	var value T
	if err := current.load(); err != nil {
		return value, err
	}
	current.mu.Lock()
	raw, ok := current.sections[name]
	current.mu.Unlock()
	if !ok {
		return value, nil
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		log.Printf("state: the %s section will not decode, it starts empty", name)
		return value, nil
	}
	return value, nil
}

// Write stores a section.
func Write[T any](name string, value T) error {
	if err := current.load(); err != nil {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	current.mu.Lock()
	current.sections[name] = raw
	current.dirty = true
	current.mu.Unlock()
	current.markDirty()
	return nil
}
