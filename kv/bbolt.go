package kv

import (
	"context"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

var topBucket = []byte("hput-kv")

// BboltKV implements KV using bbolt.
// Each path gets its own sub-bucket under the top-level "hput-kv" bucket.
type BboltKV struct {
	db *bolt.DB
}

// NewBbolt opens (or creates) a bbolt database at the given file path and
// returns a BboltKV ready for use.
func NewBbolt(file string) (*BboltKV, error) {
	db, err := bolt.Open(file, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("kv: opening bbolt db: %w", err)
	}
	// Ensure the top-level bucket exists.
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(topBucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("kv: creating top bucket: %w", err)
	}
	return &BboltKV{db: db}, nil
}

func (b *BboltKV) Close() error {
	return b.db.Close()
}

func (b *BboltKV) Get(_ context.Context, path, key string) ([]byte, error) {
	var val []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		pb := tx.Bucket(topBucket).Bucket([]byte(path))
		if pb == nil {
			return nil
		}
		v := pb.Get([]byte(key))
		if v != nil {
			val = make([]byte, len(v))
			copy(val, v)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("kv: get: %w", err)
	}
	return val, nil
}

func (b *BboltKV) Put(_ context.Context, path, key string, value []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		pb, err := tx.Bucket(topBucket).CreateBucketIfNotExists([]byte(path))
		if err != nil {
			return fmt.Errorf("kv: creating path bucket %q: %w", path, err)
		}
		return pb.Put([]byte(key), value)
	})
}

func (b *BboltKV) Delete(_ context.Context, path, key string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		pb := tx.Bucket(topBucket).Bucket([]byte(path))
		if pb == nil {
			return nil
		}
		return pb.Delete([]byte(key))
	})
}

func (b *BboltKV) List(_ context.Context, path string, opts ListOptions) (ListResult, error) {
	var result ListResult
	err := b.db.View(func(tx *bolt.Tx) error {
		pb := tx.Bucket(topBucket).Bucket([]byte(path))
		if pb == nil {
			return nil
		}

		c := pb.Cursor()
		prefix := []byte(opts.Prefix)

		// Position cursor: start from cursor token if provided, otherwise from prefix.
		var k, v []byte
		if opts.Cursor != "" {
			k, v = c.Seek([]byte(opts.Cursor))
		} else if len(prefix) > 0 {
			k, v = c.Seek(prefix)
		} else {
			k, v = c.First()
		}
		_ = v

		for ; k != nil; k, _ = c.Next() {
			if len(prefix) > 0 && !hasPrefix(k, prefix) {
				break
			}
			if opts.Limit > 0 && len(result.Keys) >= opts.Limit {
				// There are more results — encode next key as cursor.
				result.Cursor = string(k)
				return nil
			}
			result.Keys = append(result.Keys, string(k))
		}
		return nil
	})
	if err != nil {
		return ListResult{}, fmt.Errorf("kv: list: %w", err)
	}
	return result, nil
}

func hasPrefix(key, prefix []byte) bool {
	if len(key) < len(prefix) {
		return false
	}
	for i, b := range prefix {
		if key[i] != b {
			return false
		}
	}
	return true
}
