package kv

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

var (
	ErrNotStarted  = errors.New("database is not initialized")
	ErrKeyNotFound = errors.New("key not found")
)

var (
	db             *badger.DB
	mu             sync.Mutex
	openErr        error
	valueLogGCOn   sync.Once
	valueLogGCRate = 0.5
)

func EnsureStarted() error {
	mu.Lock()
	defer mu.Unlock()

	if db != nil {
		return nil
	}
	if openErr != nil {
		return openErr
	}

	baseDir, err := Dir()
	if err != nil {
		openErr = err
		return err
	}
	path := filepath.Join(baseDir, "jota", "storage", "kv")

	if err := os.MkdirAll(path, 0755); err != nil {
		openErr = err
		return err
	}

	opts := badger.DefaultOptions(path).
		WithLogger(nil).
		WithBlockCacheSize(8 << 20).
		WithMemTableSize(8 << 20).
		WithNumMemtables(1).
		WithIndexCacheSize(4 << 20).
		WithValueLogFileSize(64 << 20).
		WithCompression(options.ZSTD)
	// Open only ever publishes a usable database: Badger can answer a partial one
	// next to its error, and closing that later panics.
	d, err := badger.Open(opts)
	openErr = err
	if err != nil {
		return err
	}
	db = d
	startValueLogGC()
	return nil
}

// startValueLogGC keeps the value log compacted: every write (window state,
// cache entries, settings) appends to it, and an unconverged log makes both
// disk usage and the read path grow over time. One pass per tick, because Badger
// blocks writes while it rewrites a value log and a burst of passes is a burst
// of stalls; the next tick picks up whatever is left.
func startValueLogGC() {
	valueLogGCOn.Do(func() {
		go func() {
			for range time.Tick(5 * time.Minute) {
				mu.Lock()
				d := db
				mu.Unlock()
				if d == nil {
					return
				}
				if err := d.RunValueLogGC(valueLogGCRate); err != nil && !errors.Is(err, badger.ErrNoRewrite) {
					log.Printf("kv: value log GC: %v", err)
				}
			}
		}()
	})
}

func Start() {
	if err := EnsureStarted(); err != nil {
		log.Printf("kv: unavailable, running without persistence: %v", err)
	}
}

func Close() {
	mu.Lock()
	defer mu.Unlock()

	// A failed open is not a verdict: the next caller may be asking from a
	// different directory, so the error must not outlive the close.
	openErr = nil
	if db == nil {
		return
	}
	_ = db.Close()
	db = nil
}
