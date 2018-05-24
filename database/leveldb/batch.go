/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package leveldb

import (
	"github.com/syndtr/goleveldb/leveldb"
)

// Batch implements batch for leveldb
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

// Commit commits batch operation.
func (b *Batch) Commit() error {
	return b.leveldb.Write(b.batch, nil)
}

// Rollback rollbacks batch operation.
func (b *Batch) Rollback() {
	b.batch.Reset()
}
