/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package leveldb

import (
	"io/ioutil"
	"os"

	"github.com/seeleteam/go-seele/database"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
)

var (
	// ErrEmptyKey key is empty
	ErrEmptyKey = errors.New("key could not be empty")
)

// LevelDB wraps the leveldb
type LevelDB struct {
	db       *leveldb.DB
	quitChan chan struct{} // used by metrics
}

// NewLevelDB constructs and returns a LevelDB instance
func NewLevelDB(path string) (database.Database, error) {
	db, err := leveldb.OpenFile(path, nil)

	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(path, nil)
	}

	if err != nil {
		return nil, err
	}

	result := &LevelDB{
		db:       db,
		quitChan: make(chan struct{}),
	}

	return result, nil
}

// Close is used to close the db when not used
func (db *LevelDB) Close() {
	close(db.quitChan)
	db.db.Close()
}

// GetString gets the value for the given key
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
	if len(key) < 1 {
		return ErrEmptyKey
	}

	return db.db.Put(key, value, nil)
}

// PutString sets the value for the given key
func (db *LevelDB) PutString(key string, value string) error {
	return db.Put([]byte(key), []byte(value))
}

// Has returns true if the DB does contain the given key.
func (db *LevelDB) Has(key []byte) (ret bool, err error) {
	return db.db.Has(key, nil)
}

// HasString returns true if the DB does contain the given key.
func (db *LevelDB) HasString(key string) (ret bool, err error) {
	return db.Has([]byte(key))
}

// Delete deletes the value for the given key.
func (db *LevelDB) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

// DeleteSring deletes the value for the given key.
func (db *LevelDB) DeleteSring(key string) error {
	return db.Delete([]byte(key))
}

// NewBatch constructs and returns a batch object
func (db *LevelDB) NewBatch() database.Batch {
	batch := &Batch{
		leveldb: db.db,
		batch:   new(leveldb.Batch),
	}
	return batch
}

// NewTestDatabase creates a database instance under temp folder.
func NewTestDatabase() (db database.Database, dispose func()) {
	dir, err := ioutil.TempDir("", "Seele-LevelDB-")
	if err != nil {
		panic(err)
	}

	db, err = NewLevelDB(dir)
	if err != nil {
		os.RemoveAll(dir)
		panic(err)
	}

	return db, func() {
		db.Close()
		os.RemoveAll(dir)
	}
}
