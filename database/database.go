/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package database

// Database represents the interface of store
type Database interface {
	Close()
	Put(key []byte, value []byte) error
	Get(key []byte) ([]byte, error)
	GetString(key string) (string, error)
	PutString(key string, value string) error
	Has(key []byte) (ret bool, err error)
	HasString(key string) (ret bool, err error)
	Delete(key []byte) error
	DeleteSring(key string) error
	NewBatch() Batch
}

// Batch is the interface of batch for database
type Batch interface {
	Put(key []byte, value []byte)
	Delete(key []byte)
	Commit() error
	Rollback()
}
