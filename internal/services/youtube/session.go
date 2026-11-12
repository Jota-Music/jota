package youtube

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/Jota-Music/jota/internal/cache"
	"github.com/Jota-Music/jota/internal/music"
)

// sessionName is the file the YouTube cache lives in. With SQLite it is a key,
// not a file, but the name is kept for compatibility.
const sessionName = "youtube/session"

// token is a value with a deadline.
type token struct {
	Value string `json:"value"`
	At    int64  `json:"at"`
}

// saved is the snapshot. It is never a source of truth: an entry past its
// expiry is dropped.
type saved struct {
	Streams map[string]music.Audio `json:"streams,omitempty"`
	Visitor *token                 `json:"visitor,omitempty"`
	Account string                 `json:"account,omitempty"`
	Version *token                 `json:"version,omitempty"`
	Client  string                 `json:"client,omitempty"`
}

// sessionCache is the YouTube cache: held in memory, written whole when it
// changes.
type sessionCache struct {
	mu     sync.Mutex
	loaded bool
	saved  saved
}

var youtubeCache = &sessionCache{}

func (c *sessionCache) load() {
	if c.loaded {
		return
	}
	c.loaded = true

	data, ok := cache.ReadRaw(sessionName)
	if !ok {
		return
	}
	var stored saved
	if err := json.Unmarshal(data, &stored); err != nil {
		return
	}
	c.saved = stored
}

func (c *sessionCache) write() {
	data, err := json.Marshal(c.saved)
	if err != nil {
		return
	}
	_ = cache.Write(sessionName, data)
}

func (c *sessionCache) stream(key string) (music.Audio, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	audio, ok := c.saved.Streams[key]
	if !ok {
		return music.Audio{}, false
	}
	if !audioCacheStillValid(&audio) {
		return music.Audio{}, false
	}
	return audio, true
}

func (c *sessionCache) keep(key string, audio music.Audio) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	if c.saved.Streams == nil {
		c.saved.Streams = map[string]music.Audio{}
	}
	now := time.Now().UTC().Unix()
	for stored, entry := range c.saved.Streams {
		if entry.ExpireAt <= now {
			delete(c.saved.Streams, stored)
		}
	}
	c.saved.Streams[key] = audio
	c.write()
}

func (c *sessionCache) forget(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	if _, ok := c.saved.Streams[key]; !ok {
		return
	}
	delete(c.saved.Streams, key)
	c.write()
}

func (c *sessionCache) visitor() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	if c.saved.Visitor == nil {
		return ""
	}
	if time.Since(time.UnixMilli(c.saved.Visitor.At)) >= visitorPersistTTL {
		return ""
	}
	return c.saved.Visitor.Value
}

func (c *sessionCache) rememberVisitor(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	c.saved.Visitor = &token{Value: value, At: time.Now().UnixMilli()}
	c.write()
}

func (c *sessionCache) forgetVisitor() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	if c.saved.Visitor == nil {
		return
	}
	c.saved.Visitor = nil
	c.write()
}

func (c *sessionCache) accountVisitor() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	return c.saved.Account
}

func (c *sessionCache) rememberAccountVisitor(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	if c.saved.Account == value {
		return
	}
	c.saved.Account = value
	c.write()
}

func (c *sessionCache) forgetAccountVisitor() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	if c.saved.Account == "" {
		return
	}
	c.saved.Account = ""
	c.write()
}

func (c *sessionCache) version() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	if c.saved.Version == nil {
		return ""
	}
	if time.Since(time.UnixMilli(c.saved.Version.At)) >= versionTTL {
		return ""
	}
	return c.saved.Version.Value
}

func (c *sessionCache) rememberVersion(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	c.saved.Version = &token{Value: value, At: time.Now().UnixMilli()}
	c.write()
}

func (c *sessionCache) client() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	return c.saved.Client
}

func (c *sessionCache) rememberClient(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	if c.saved.Client == name {
		return
	}
	c.saved.Client = name
	c.write()
}

func (c *sessionCache) forgetClient() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()

	if c.saved.Client == "" {
		return
	}
	c.saved.Client = ""
	c.write()
}
