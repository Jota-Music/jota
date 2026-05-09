package kv

import (
	"encoding/json"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type Bucket struct {
	prefix string
}

func UseBucket(name string) *Bucket {
	return &Bucket{prefix: name + ":"}
}

func (b *Bucket) key(key string) []byte {
	return []byte(b.prefix + key)
}

func (b *Bucket) SetObject(key string, value any, ttl ...time.Duration) error {
	if err := EnsureStarted(); err != nil {
		return err
	}
	if err := checkDB(); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var d time.Duration
	if len(ttl) > 0 {
		d = ttl[0]
	}
	return db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(b.key(key), data)
		if d > 0 {
			e = e.WithTTL(d)
		}
		return txn.SetEntry(e)
	})
}

func (b *Bucket) Delete(key string) error {
	if err := EnsureStarted(); err != nil {
		return err
	}
	if err := checkDB(); err != nil {
		return err
	}
	return db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(b.key(key))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		return err
	})
}

func (b *Bucket) GetObject(key string, out any) error {
	if err := EnsureStarted(); err != nil {
		return err
	}
	if err := checkDB(); err != nil {
		return err
	}
	var data []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(b.key(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return KeyNotFoundError
			}
			return err
		}
		data, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func (b *Bucket) SetString(key string, value string, ttl ...time.Duration) error {
	if err := EnsureStarted(); err != nil {
		return err
	}
	if err := checkDB(); err != nil {
		return err
	}
	data := []byte(value)
	var d time.Duration
	if len(ttl) > 0 {
		d = ttl[0]
	}
	return db.Update(func(txn *badger.Txn) error {
		e := badger.NewEntry(b.key(key), data)
		if d > 0 {
			e = e.WithTTL(d)
		}
		return txn.SetEntry(e)
	})
}

func (b *Bucket) GetString(key string) (string, error) {
	if err := EnsureStarted(); err != nil {
		return "", err
	}
	if err := checkDB(); err != nil {
		return "", err
	}
	var data []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(b.key(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return KeyNotFoundError
			}
			return err
		}
		data, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return "", err
	}
	return string(data), nil
}
