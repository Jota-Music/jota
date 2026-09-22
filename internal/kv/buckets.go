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

func (b *Bucket) set(key string, data []byte, ttl ...time.Duration) error {
	if err := EnsureStarted(); err != nil {
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
	return db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(b.key(key))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		return err
	})
}

func (b *Bucket) get(key string) ([]byte, error) {
	if err := EnsureStarted(); err != nil {
		return nil, err
	}
	var data []byte
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(b.key(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrKeyNotFound
			}
			return err
		}
		data, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ForEach visits every stored entry in the bucket, passing the bare key and
// raw value. Used by refresh loops that revalidate cached content.
func (b *Bucket) ForEach(fn func(key string, data []byte) error) error {
	if err := EnsureStarted(); err != nil {
		return err
	}
	prefix := []byte(b.prefix)
	return db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := string(item.KeyCopy(nil))[len(b.prefix):]
			data, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			if err := fn(key, data); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *Bucket) SetObject(key string, value any, ttl ...time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return b.set(key, data, ttl...)
}

func (b *Bucket) GetObject(key string, out any) error {
	data, err := b.get(key)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func (b *Bucket) SetString(key string, value string, ttl ...time.Duration) error {
	return b.set(key, []byte(value), ttl...)
}

func (b *Bucket) GetString(key string) (string, error) {
	data, err := b.get(key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
