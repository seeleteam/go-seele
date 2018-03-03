/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package leveldb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

type LevelDB struct {
	db *leveldb.DB
}

func NewLevelDB(path string) (*LevelDB, error) {
	db, err := leveldb.OpenFile(path, nil)

	if err != nil {
		if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
			db, err = leveldb.RecoverFile(path, nil)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	result := &LevelDB{
		db: db,
	}

	return result, nil
}

// Close don't forget close db when not use
func (db *LevelDB) Close() {
	db.db.Close()
}

// Get gets the value for the given key
func (db *LevelDB) GetString(key string) (string, error) {
	value, err := db.Get([]byte(key))

	return string(value), err
}

// Get gets the value for the given key
func (db *LevelDB) Get(key []byte) ([]byte, error) {
	return db.db.Get(key, nil)
}

// Put sets the value for the given key
func (db *LevelDB) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

// Put sets the value for the given key
func (db *LevelDB) PutString(key string, value string) error {
	return db.Put([]byte(key), []byte(value))
}

// Has returns true if the DB does contains the given key.
func (db *LevelDB) Has(key []byte) (ret bool, err error) {
	return db.db.Has(key, nil)
}

// Has returns true if the DB does contains the given key.
func (db *LevelDB) HasString(key string) (ret bool, err error) {
	return db.Has([]byte(key))
}

// Delete deletes the value for the given key.
func (db *LevelDB) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

// Delete deletes the value for the given key.
func (db *LevelDB) DeleteSring(key string) error {
	return db.Delete([]byte(key))
}
