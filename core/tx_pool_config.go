/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

// TransactionPoolConfig is the configuration of transaction pool.
type TransactionPoolConfig struct {
	MaximumTransactions uint
}

// DefaultTxPoolConfig returns the default configuration of transaction pool.
func DefaultTxPoolConfig() *TransactionPoolConfig {
	return &TransactionPoolConfig{
		MaximumTransactions: 1024,
	}
}
