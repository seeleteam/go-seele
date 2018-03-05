/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package leveldb

import (
	"github.com/seeleteam/go-seele/database"

	"github.com/syndtr/goleveldb/leveldb"
)

// Batch batch  implementation for leveldb
type Batch struct {
	db      database.Database
	leveldb *leveldb.DB
	batch   *leveldb.Batch
}

// Put sets the value for the given key
func (b *Batch) Put(key []byte, value []byte) {
	b.batch.Put(key, value)
}

// Delete deletes the value for the given key.
func (b *Batch) Delete(key []byte) {
	b.batch.Delete(key)
}

// Commit commit batch operator.
func (b *Batch) Commit() error {
	return b.leveldb.Write(b.batch, nil)
}

// Rollback rollback batch operator.
func (b *Batch) Rollback() {
	b.batch.Reset()
}
