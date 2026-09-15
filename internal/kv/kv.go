package kv

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

var (
	ErrNotStarted  = errors.New("database is not initialized")
	ErrKeyNotFound = errors.New("key not found")
)

var (
	db      *badger.DB
	mu      sync.Mutex
	openErr error
)

func getDatabasePath() (string, error) {
	baseDir, err := storageBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "jota", "storage", "kv"), nil
}

func EnsureStarted() error {
	mu.Lock()
	defer mu.Unlock()

	if db != nil {
		return nil
	}
	if openErr != nil {
		return openErr
	}

	path, err := getDatabasePath()
	if err != nil {
		openErr = err
		return err
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		openErr = err
		return err
	}

	opts := badger.DefaultOptions(path).
		WithLogger(nil).
		WithBlockCacheSize(8 << 20).
		WithMemTableSize(8 << 20).
		WithNumMemtables(1)
	db, openErr = badger.Open(opts)
	return openErr
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

func checkDB() error {
	if db == nil {
		return ErrNotStarted
	}
	return nil
}
