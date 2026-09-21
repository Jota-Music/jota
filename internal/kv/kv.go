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

	baseDir, err := storageBaseDir()
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
	db, openErr = badger.Open(opts)
	if openErr == nil {
		startValueLogGC()
	}
	return openErr
}

// startValueLogGC keeps the value log compacted: every write (window state,
// cache entries, settings) appends to it, and an unconverged log makes both
// disk usage and the read path grow over time. Each RunValueLogGC pass handles
// one rewrite, so keep retrying while Badger reports progress and only stop on
// ErrNoRewrite.
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
				for {
					err := d.RunValueLogGC(valueLogGCRate)
					if err != nil {
						if !errors.Is(err, badger.ErrNoRewrite) {
							log.Printf("kv: value log GC: %v", err)
						}
						break
					}
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

	if db == nil {
		return
	}
	_ = db.Close()
	db = nil
	openErr = nil
}
