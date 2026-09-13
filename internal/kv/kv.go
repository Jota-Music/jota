package kv

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

// Errores personalizados
var (
	KvNotStartedError = errors.New("database is not initialized")
	KeyNotFoundError  = errors.New("key not found")
)

var (
	db        *badger.DB
	startOnce sync.Once
	startErr  error
)

func getDatabasePath() (string, error) {
	baseDir, err := storageBaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, "jota", "storage", "kv"), nil
}

func EnsureStarted() error {
	startOnce.Do(func() {
		if db != nil {
			return
		}

		path, err := getDatabasePath()
		if err != nil {
			startErr = err
			return
		}

		err = os.MkdirAll(path, 0755)
		if err != nil {
			startErr = err
			return
		}

		opts := badger.DefaultOptions(path).WithLogger(nil)
		db, startErr = badger.Open(opts)
	})
	return startErr
}

func Start() {
	if err := EnsureStarted(); err != nil {
		log.Fatal(err)
	}
}

func Close() {
	if db != nil {
		db.Close()
	}
}

func checkDB() error {
	if db == nil {
		return KvNotStartedError
	}
	return nil
}
