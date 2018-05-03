/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package leveldb

import (
	"github.com/syndtr/goleveldb/leveldb"
)

// Batch batch implementation for leveldb
type Batch struct {
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

// Commit commits batch operator.
func (b *Batch) Commit() error {
	return b.leveldb.Write(b.batch, nil)
}

// Rollback rollbacks batch operator.
func (b *Batch) Rollback() {
	b.batch.Reset()
}
