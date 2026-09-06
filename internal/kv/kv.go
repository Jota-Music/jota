package kv

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

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
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "jota", "storage", "kv"), nil
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

func New(key string, value string, ttl time.Duration) error {
	if err := checkDB(); err != nil {
		return err
	}
	return db.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), []byte(value))
		if ttl > 0 {
			entry = entry.WithTTL(ttl)
		}
		return txn.SetEntry(entry)
	})
}

func Delete(key string) error {
	if err := checkDB(); err != nil {
		return err
	}
	return db.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		if err == badger.ErrKeyNotFound {
			return KeyNotFoundError
		}
		return err
	})
}

func Get(key string) (string, error) {
	if err := checkDB(); err != nil {
		return "", err
	}
	var valCopy []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return KeyNotFoundError
			}
			return err
		}
		valCopy, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return "", err
	}
	return string(valCopy), nil
}
