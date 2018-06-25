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
		// 1 simple transaction is about 152 byte size. So 1000 transactions is about 152kb. And 10000 transaction is about 1.52Mb.
		// We want to use 10MB memory for tx pool. so it is about 520k transactions.
		// So we could cache transactions for about 100 blocks, which means at least 25 minutes block generation consume
		Capacity: 500000,
	}
}
