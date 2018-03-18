/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"errors"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

var (
	errTxHashExists = errors.New("transaction hash already exists")
	errTxPoolFull   = errors.New("transaction pool is full")
)

// TransactionPool is a thread-safe container for transactions that received
// from the network or submitted locally. A transaction will be removed from
// the pool once included in a blockchain.
type TransactionPool struct {
	mutex       sync.RWMutex
	config      TransactionPoolConfig
	hashToTxMap map[common.Hash]*types.Transaction
}

// NewTransactionPool creates and returns a transaction pool.
func NewTransactionPool(config TransactionPoolConfig) *TransactionPool {
	return &TransactionPool{
		config:      config,
		hashToTxMap: make(map[common.Hash]*types.Transaction),
	}
}

// AddTransaction adds a single transation into the pool if it is valid and return true.
// Otherwise, return false and concrete error.
func (pool *TransactionPool) AddTransaction(tx *types.Transaction) (bool, error) {
	if err := tx.Validate(); err != nil {
		return false, err
	}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if pool.hashToTxMap[tx.Hash] != nil {
		return false, errTxHashExists
	}

	if uint(len(pool.hashToTxMap)) >= pool.config.Capacity {
		return false, errTxPoolFull
	}

	pool.hashToTxMap[tx.Hash] = tx

	return true, nil
}

// Pending returns the pending transactions in the transaction pool.
func (pool *TransactionPool) Pending() ([]*types.Transaction, error) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	// TODO
	return nil,nil
}
