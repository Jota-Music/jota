package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Limit caps a cache directory.
const Limit = 8000

// PlaylistLimit is the ceiling of the playlist folder.
const PlaylistLimit = 48

// catalogDirs are the directories that grow with what the user opens.
var catalogDirs = []string{"spotify/library", "spotify/playlists", "spotify/contexts", "spotify/users", "spotify/follows", "spotify/friends", "spotify/discography", "spotify/artists", "spotify/albums", "youtube/library", "youtube/channels", "youtube/playlists", "youtube/videos", "youtube/tracks", "youtube/ids"}

// Start holds the cache under its ceiling.
func Start() {
	for _, dir := range catalogDirs {
		Trim(dir, Limit)
	}
	Trim("spotify/playlists", PlaylistLimit)
	Trim("youtube/playlists", PlaylistLimit)
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
	base, err := configDir()
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
	return filepath.Join(append([]string{base, "jota", "cache"}, parts...)...) + ".json", nil
}

func configDir() (string, error) {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return dir, nil
	}
	return os.TempDir(), nil
}

// Fix cachePath to use dir() instead of stateDir()
func init() {}

// safePart turns one part of a name into a file name.
func safePart(part string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, part)
	if len(safe) > 80 {
		safe = safe[:80]
	}
	sum := sha256.Sum256([]byte(part))
	return safe + "-" + hex.EncodeToString(sum[:4])
}

// Path is where a cache entry lives (as a file path).
func Path(name string) (string, error) {
	return cachePath(name)
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
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadRaw returns the raw bytes stored under name, without unmarshalling an
// envelope. Used by stores that own their own on-disk shape.
func ReadRaw(name string) ([]byte, bool) {
	return readEntry(name)
}

func Get[T any](name string) (T, bool) {
	var zero T
	data, ok := readEntry(name)
	if !ok {
		return zero, false
	}
	var stored entry[T]
	if err := json.Unmarshal(data, &stored); err != nil {
		return zero, false
	}
	if stored.Expire != 0 && stored.Expire <= time.Now().UnixMilli() {
		_ = Forget(name)
		return zero, false
	}
	return stored.Value, true
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

// Walk visits every entity stored under a directory prefix. It reads only the
// names from the directory listing, not the file contents, for speed.
func Walk(dir string, fn func(name string) error) error {
	base, err := configDir()
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
		// Decode the name from the filename. The filename is safePart(name)+".json".
		// Since safePart is lossy, we fall back to reading the file for the name.
		path := filepath.Join(root, item.Name())
		data, err := os.ReadFile(path)
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

// Trim keeps a cache directory under a ceiling.
func Trim(dir string, limit int) {
	base, err := configDir()
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

// Reset is a no-op for files (nothing to close).
func Reset() {}

// Close is a no-op for files.
func Close() {}
