package cache

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Jota-Music/jota/internal/state"
)

// Limit caps a cache directory.
const Limit = 8000

// PlaylistLimit is the ceiling of the playlist folder.
const PlaylistLimit = 48

// started is when the process began. A temporary file older than it belongs to
// a write that a crash cut short, and nothing else would ever collect it.
var started = time.Now()

// limits is the ceiling of each folder that grows with what the user opens. A
// folder that is not here is never trimmed: the picks under youtube/links are
// 11 bytes each, and counting 8000 of them would cost more than the space.
// An entry that is a single file (the session, the account's own library) lives
// next to the folders and has nothing to trim.
var limits = map[string]int{
	"spotify/albums":      Limit,
	"spotify/artists":     Limit,
	"spotify/contexts":    Limit,
	"spotify/discography": Limit,
	"spotify/follows":     Limit,
	"spotify/friends":     Limit,
	"spotify/library":     Limit,
	"spotify/playlists":   PlaylistLimit,
	"spotify/tracks":      Limit,
	"spotify/users":       Limit,
	"youtube/channels":    Limit,
	"youtube/ids":         Limit,
	"youtube/playlists":   PlaylistLimit,
	"youtube/streams":     Limit,
	"youtube/tracks":      Limit,
	"youtube/videos":      Limit,
}

// Start holds the cache under its ceiling and clears what a crash left behind.
func Start() {
	sweep()
	for dir, limit := range limits {
		Trim(dir, limit)
	}
}

// sweep removes the temporary files a crash left behind. A write that died
// between the temporary file and the rename leaves it, and nothing else would
// ever collect it. A temporary file this run wrote is still in flight, so it
// stays.
func sweep() {
	base, err := state.Dir()
	if err != nil {
		return
	}
	root := filepath.Join(base, "jota", "cache")
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".tmp") {
			return nil
		}
		info, err := d.Info()
		if err == nil && info.ModTime().Before(started) {
			_ = os.Remove(path)
		}
		return nil
	})
}

// entry is what a cached entity looks like on disk.
type entry[T any] struct {
	Name        string `json:"name"`
	Saved       int64  `json:"saved"`
	Expire      int64  `json:"expire,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	Value       T      `json:"value"`
}

// cachePath is where an entity lives: the config folder, under jota/cache/,
// under the name the caller chose.
func cachePath(name string) (string, error) {
	return cacheLocation(name, ".json")
}

// RawPath is the on-disk location of a cache entry that owns its own shape and
// does not want the cache's .json suffix.
func RawPath(name string) (string, error) {
	return cacheLocation(name, "")
}

func cacheLocation(name, suffix string) (string, error) {
	base, err := state.Dir()
	if err != nil {
		return "", err
	}
	parts := strings.Split(name, "/")
	for i, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", errors.New("cache: invalid name " + name)
		}
		if i == len(parts)-1 {
			parts[i] = safePart(part)
		}
	}
	return filepath.Join(append([]string{base, "jota", "cache"}, parts...)...) + suffix, nil
}

// safePart turns one part of a name into a file name, and the name is
// recoverable from it: the base64 of the part when it fits, a hash when it does
// not, so walking a directory does not have to open every file to learn what it
// holds.
func safePart(part string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(part))
	if len(encoded) <= 80 {
		return "~" + encoded
	}
	sum := sha256.Sum256([]byte(part))
	return "_" + hex.EncodeToString(sum[:16])
}

// partOf reads back the name a file name was made from. A name the encoder
// could not carry (one longer than it fits) answers false, and the caller falls
// back to the name stored inside the file.
func partOf(file string) (string, bool) {
	if !strings.HasPrefix(file, "~") {
		return "", false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(file[1:])
	if err != nil {
		return "", false
	}
	return string(decoded), true
}

// Write replaces a cache file through a temporary one.
func Write(name string, data []byte) error {
	return writeEntry(name, data)
}

func readEntry(name string) ([]byte, bool) {
	path, err := cachePath(name)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

func writeEntry(name string, data []byte) error {
	path, err := cachePath(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// A temporary name of its own per write: two writers of the same entity (a
	// background refresh and the request that asked for it) would otherwise share
	// one file and publish a mixture of both.
	suffix := make([]byte, 4)
	if _, err := rand.Read(suffix); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%x.tmp", path, suffix)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// ReadRaw returns the raw bytes stored under name, without unmarshalling an
// envelope. Used by stores that own their own on-disk shape.
func ReadRaw(name string) ([]byte, bool) {
	return readEntry(name)
}

func Get[T any](name string) (T, bool) {
	value, _, ok := GetWithMeta[T](name)
	return value, ok
}

// GetWithMeta is Get plus the moment the entry was written. A caller that can
// tell whether the value was proven recently (a stream's URL was just probed)
// uses it to skip a check without an extra read: the envelope is already here.
func GetWithMeta[T any](name string) (T, int64, bool) {
	var zero T
	data, ok := readEntry(name)
	if !ok {
		return zero, 0, false
	}
	var stored entry[T]
	if err := json.Unmarshal(data, &stored); err != nil {
		return zero, 0, false
	}
	if stored.Expire != 0 && stored.Expire <= time.Now().UnixMilli() {
		_ = Forget(name)
		return zero, 0, false
	}
	return stored.Value, stored.Saved, true
}

func Put[T any](name string, value T, ttl ...time.Duration) error {
	stored := entry[T]{
		Name:        name,
		Saved:       time.Now().UnixMilli(),
		Fingerprint: Fingerprint(value),
		Value:       value,
	}
	if len(ttl) > 0 && ttl[0] > 0 {
		stored.Expire = time.Now().Add(ttl[0]).UnixMilli()
	}
	data, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	return writeEntry(name, data)
}

func Forget(name string) error {
	path, err := cachePath(name)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func Cached[T any](name string, fetch func() (T, error)) (T, error) {
	if value, ok := Get[T](name); ok {
		return value, nil
	}
	value, err := fetch()
	if err != nil {
		var zero T
		return zero, err
	}
	_ = Put(name, value)
	return value, nil
}

func Refreshable[T any](name string, fetch func() (T, error)) bool {
	data, ok := readEntry(name)
	if !ok {
		return false
	}
	var stored entry[T]
	if err := json.Unmarshal(data, &stored); err != nil {
		return false
	}
	if stored.Fingerprint == "" {
		return false
	}
	value, err := fetch()
	if err != nil {
		return false
	}
	fingerprint := Fingerprint(value)
	if fingerprint == stored.Fingerprint {
		return false
	}
	_ = Put(name, value)
	return true
}

// Walk visits every entity stored under a directory prefix. The name comes out
// of the file name, which carries it, so a pass over a folder of thousands of
// entries does not read or decode a single file. An entry written by the older
// naming form still carries the name inside it and is read for that.
func Walk(dir string, fn func(name string) error) error {
	base, err := state.Dir()
	if err != nil {
		return err
	}
	root := filepath.Join(base, "jota", "cache", dir)
	items, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, item := range items {
		if item.IsDir() || !strings.HasSuffix(item.Name(), ".json") {
			continue
		}
		if part, ok := partOf(strings.TrimSuffix(item.Name(), ".json")); ok {
			if err := fn(dir + "/" + part); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, item.Name()))
		if err != nil {
			continue
		}
		var stored struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &stored); err != nil || stored.Name == "" {
			continue
		}
		if err := fn(stored.Name); err != nil {
			return err
		}
	}
	return nil
}

// Sweep drops the entries of a directory that are older than the given age.
// It is how the entries with a short life are collected without reading them:
// a file that old cannot be alive any more. It answers how many it dropped.
func Sweep(dir string, olderThan time.Duration) int {
	base, err := state.Dir()
	if err != nil {
		return 0
	}
	root := filepath.Join(base, "jota", "cache", dir)
	items, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	cutoff := time.Now().Add(-olderThan)
	dropped := 0
	for _, item := range items {
		if item.IsDir() || !strings.HasSuffix(item.Name(), ".json") {
			continue
		}
		info, err := item.Info()
		if err != nil || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(root, item.Name())); err == nil {
			dropped++
		}
	}
	return dropped
}

// Trim keeps a cache directory under a ceiling.
func Trim(dir string, limit int) {
	base, err := state.Dir()
	if err != nil {
		return
	}
	root := filepath.Join(base, "jota", "cache", dir)
	items, err := os.ReadDir(root)
	if err != nil {
		return
	}
	type aged struct {
		name string
		at   time.Time
	}
	var entries []aged
	for _, item := range items {
		if item.IsDir() || !strings.HasSuffix(item.Name(), ".json") {
			continue
		}
		info, err := item.Info()
		if err != nil {
			continue
		}
		entries = append(entries, aged{item.Name(), info.ModTime()})
	}
	if len(entries) <= limit {
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].at.Before(entries[j].at) })
	for _, e := range entries[:len(entries)-limit] {
		_ = os.Remove(filepath.Join(root, e.name))
	}
}

// Fingerprint hashes a value.
func Fingerprint(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
