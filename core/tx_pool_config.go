/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

// TransactionPoolConfig is the configuration of transaction pool.
type TransactionPoolConfig struct {
	Capacity uint // Maximum number of transactions in pool.
}

// DefaultTxPoolConfig returns the default configuration of transaction pool.
func DefaultTxPoolConfig() *TransactionPoolConfig {
	return &TransactionPoolConfig{
		Capacity: 1024,
	}
}
