/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

// TransactionPoolConfig is the configuration of the transaction pool.
type TransactionPoolConfig struct {
	Capacity uint // Maximum number of transactions in the pool.
}

// DefaultTxPoolConfig returns the default configuration of the transaction pool.
func DefaultTxPoolConfig() *TransactionPoolConfig {
	return &TransactionPoolConfig{
		// 1 simple transaction is about 152 byte size. So 1000 transactions is about 152KB, and 10000 transaction is about 1.52MB.
		// We want to cache transactions for about 100 blocks (about 500k transactions), which means at least 25 minutes block generation consume,
		// the memory usage will be <=100MB for tx pool.
		Capacity: 500000,
	}
}
