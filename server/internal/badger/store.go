// Package badger wraps the embedded Badger key-value store: the operational
// database of the platform (catalog, clients, orders, campaigns...).
package badger

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	badgerdb "github.com/dgraph-io/badger/v4"
)

type Store struct {
	db *badgerdb.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return nil, err
	}
	opts := badgerdb.DefaultOptions(path)
	opts.Logger = nil // silent in production; errors surface via returns
	db, err := badgerdb.Open(opts)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) DB() *badgerdb.DB { return s.db }

// PutJSON marshals v and stores it under key.
func (s *Store) PutJSON(key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set([]byte(key), b)
	})
}

// GetJSON reads key into v. Returns os.ErrNotExist when missing.
func (s *Store) GetJSON(key string, v any) error {
	return s.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get([]byte(key))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return os.ErrNotExist
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, v)
		})
	})
}

func (s *Store) GetString(key string) (string, error) {
	var out string
	err := s.db.View(func(txn *badgerdb.Txn) error {
		item, err := txn.Get([]byte(key))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			return os.ErrNotExist
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			out = string(val)
			return nil
		})
	})
	return out, err
}

func (s *Store) PutString(key, val string) error {
	return s.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Set([]byte(key), []byte(val))
	})
}

func (s *Store) Delete(key string) error {
	return s.db.Update(func(txn *badgerdb.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// ScanPrefix iterates all keys with prefix, unmarshalling JSON values into a new T each time.
// Iteration order is ascending by key. Return false from fn to stop early.
func (s *Store) ScanPrefix(prefix string, v any, fn func(key string) bool) error {
	return s.db.View(func(txn *badgerdb.Txn) error {
		opts := badgerdb.DefaultIteratorOptions
		opts.Prefix = []byte(prefix) // Rewind() seeks to the first key with this prefix
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			key := string(item.KeyCopy(nil))
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, v)
			}); err != nil {
				return err
			}
			if !fn(key) {
				return nil
			}
		}
		return nil
	})
}

// Batch executes writes atomically.
func (s *Store) Batch(fn func(txn *badgerdb.Txn) error) error {
	return s.db.Update(fn)
}

// NextFolio returns the next order sequence number (meta:seq:folio), creating it if absent.
func (s *Store) NextFolio() (int64, error) {
	var n int64
	err := s.db.Update(func(txn *badgerdb.Txn) error {
		item, err := txn.Get([]byte(PrefMeta + MetaFolioSeq))
		if errors.Is(err, badgerdb.ErrKeyNotFound) {
			n = 1
		} else if err != nil {
			return err
		} else {
			if err := item.Value(func(val []byte) error {
				_, err := fmt.Sscanf(string(val), "%d", &n)
				return err
			}); err != nil {
				return err
			}
			n++
		}
		return txn.Set([]byte(PrefMeta+MetaFolioSeq), []byte(fmt.Sprintf("%d", n)))
	})
	return n, err
}

// Backup streams a consistent snapshot of the whole store into a single file
// at dataDir/backups/badger-YYYYMMDD-HHMMSS.bak (restorable with db.Load()).
func (s *Store) Backup(dataDir string) (string, error) {
	dir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "badger-"+time.Now().Format("20060102-150405")+".bak")
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := s.db.Backup(f, 0); err != nil {
		return "", err
	}
	return path, nil
}
