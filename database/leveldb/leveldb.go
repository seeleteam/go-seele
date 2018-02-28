/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package leveldb

import (
	"github.com/syndtr/goleveldb/leveldb"
)

type levelDB struct {
	db *leveldb.DB
}

func NewLevelDB(path string) (*levelDB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	result := &levelDB{
		db: db,
	}

	return result, nil
}

// Close don't forget close db when not use
func (db *levelDB) Close() {
	db.db.Close()
}

func (db *levelDB) GetString(key string) (string, error) {
	value, err := db.Get([]byte(key))

	return string(value), err
}

func (db *levelDB) Get(key []byte) ([]byte, error) {
	return db.db.Get(key, nil)
}

func (db *levelDB) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

func (db *levelDB) PutString(key string, value string) error {
	return db.Put([]byte(key), []byte(value))
}

func (db *levelDB) Has(key []byte) (ret bool, err error) {
	return db.db.Has(key, nil)
}

func (db *levelDB) HasString(key string) (ret bool, err error) {
	return db.Has([]byte(key))
}

func (db *levelDB) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

func (db *levelDB) DeleteSring(key string) error {
	return db.Delete([]byte(key))
}
