/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package leveldb

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/seeleteam/go-seele/database"
)

/*
 * This is a test memory database. Do not use for any production it does not get persisted
 */
type MemDatabase struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewMemDatabase() database.Database {
	return &MemDatabase{
		db: make(map[string][]byte),
	}
}

func (db *MemDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

// PutString sets the value for the given key
func (db *MemDatabase) PutString(key string, value string) error {
	return db.Put([]byte(key), []byte(value))
}

func (db *MemDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[string(key)]
	return ok, nil
}

// HasString returns true if the DB does contain the given key.
func (db *MemDatabase) HasString(key string) (ret bool, err error) {
	return db.Has([]byte(key))
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, errors.New("not found")
}

// GetString gets the value for the given key
func (db *MemDatabase) GetString(key string) (string, error) {
	value, err := db.Get([]byte(key))

	return string(value), err
}

func (db *MemDatabase) Keys() [][]byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := [][]byte{}
	for key := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

func (db *MemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}

// DeleteSring deletes the value for the given key.
func (db *MemDatabase) DeleteSring(key string) error {
	return db.Delete([]byte(key))
}

func (db *MemDatabase) Close() {}

// NewBatch constructs and returns a batch object
func (db *MemDatabase) NewBatch() database.Batch {
	panic("unsupported")
}
