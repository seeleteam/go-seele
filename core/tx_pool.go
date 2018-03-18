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
	mutex           sync.RWMutex
	config          TransactionPoolConfig
	hashToTxMap     map[common.Hash]*types.Transaction
	accountToTxsMap map[common.Address]*txCollection // Account address to tx collection mapping.
}

// NewTransactionPool creates and returns a transaction pool.
func NewTransactionPool(config TransactionPoolConfig) *TransactionPool {
	pool := &TransactionPool{
		config:          config,
		hashToTxMap:     make(map[common.Hash]*types.Transaction),
		accountToTxsMap: make(map[common.Address]*txCollection),
	}

	// TODO register event listeners

	return pool
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

	if _, ok := pool.accountToTxsMap[tx.Data.From]; !ok {
		pool.accountToTxsMap[tx.Data.From] = newTxCollection()
	}

	pool.accountToTxsMap[tx.Data.From].add(tx)

	return true, nil
}

// GetTransaction returns a transaction if it is contained in the pool and nil otherwise.
func (pool *TransactionPool) GetTransaction(txHash common.Hash) *types.Transaction {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	return pool.hashToTxMap[txHash]
}

// GetProcessableTransactions retrieves all processable transactions. The returned transactions
// are grouped by origin account address and sorted by nonce ASC.
func (pool *TransactionPool) GetProcessableTransactions() map[common.Address][]*types.Transaction {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	allAccountTxs := make(map[common.Address][]*types.Transaction)

	for account, txs := range pool.accountToTxsMap {
		allAccountTxs[account] = txs.getTxsOrderByNonceAsc()
	}

	return allAccountTxs
}

// Stop terminates the transaction pool.
func (pool *TransactionPool) Stop() {
	// TODO remove event listeners
}

// Pending returns the pending transactions in the transaction pool.
func (pool *TransactionPool) Pending() ([]*types.Transaction, error) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	// TODO
	return nil, nil
}
