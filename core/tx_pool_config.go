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
		Capacity: 10000, // 1 simple transaction is about 152 byte size. So 1000 transactions is about 1.28MB. And 10000 transaction is about 12.8MB
	}
}
