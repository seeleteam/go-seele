/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"errors"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
)

var (
	errTxHashExists = errors.New("transaction hash already exists")
	errTxPoolFull   = errors.New("transaction pool is full")
	errTxFeeNil     = errors.New("fee can't be nil")
	errTxNonceUsed  = errors.New("transaction from this address already used its nonce")
)

// The status of transaction in tx pool
const (
	PENDING    byte = 0x01
	PROCESSING byte = 0x02
	ERROR      byte = 0x04
	ALL        byte = PENDING | PROCESSING | ERROR
)

type blockchain interface {
	CurrentState() *state.Statedb
	GetStore() store.BlockchainStore
}

type pooledTx struct {
	transaction *types.Transaction
	txStatus    byte
}

// TransactionPool is a thread-safe container for transactions received
// from the network or submitted locally. A transaction will be removed from
// the pool once included in a blockchain.
type TransactionPool struct {
	mutex           sync.RWMutex
	config          TransactionPoolConfig
	chain           blockchain
	hashToTxMap     map[common.Hash]*pooledTx
	accountToTxsMap map[common.Address]*txCollection // Account address to tx collection mapping.
}

// NewTransactionPool creates and returns a transaction pool.
func NewTransactionPool(config TransactionPoolConfig, chain blockchain) *TransactionPool {
	pool := &TransactionPool{
		config:          config,
		chain:           chain,
		hashToTxMap:     make(map[common.Hash]*pooledTx),
		accountToTxsMap: make(map[common.Address]*txCollection),
	}

	return pool
}

// AddTransaction adds a single transaction into the pool if it is valid and returns nil.
// Otherwise, return the concrete error.
func (pool *TransactionPool) AddTransaction(tx *types.Transaction) error {
	statedb := pool.chain.CurrentState()
	if err := tx.Validate(statedb); err != nil {
		return err
	}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if pool.hashToTxMap[tx.Hash] != nil {
		return errTxHashExists
	}

	if uint(len(pool.hashToTxMap)) >= pool.config.Capacity {
		return errTxPoolFull
	}

	if tx.Data.Fee == nil {
		return errTxFeeNil
	}

	existTx := pool.findTransaction(tx.Data.From, tx.Data.AccountNonce, PENDING)
	if existTx != nil {
		if tx.Data.Fee.Cmp(existTx.Data.Fee) > 0 {
			pool.removeTransaction(existTx.Hash)
		} else {
			return errTxNonceUsed
		}
	}

	pool.addTransaction(tx)

	// fire event
	event.TransactionInsertedEventManager.Fire(tx)

	return nil
}

func (pool *TransactionPool) addTransaction(tx *types.Transaction) {
	poolTx := &pooledTx{tx, PENDING}
	pool.hashToTxMap[tx.Hash] = poolTx

	if _, ok := pool.accountToTxsMap[tx.Data.From]; !ok {
		pool.accountToTxsMap[tx.Data.From] = newTxCollection()
	}

	pool.accountToTxsMap[tx.Data.From].add(pool.hashToTxMap[tx.Hash])
}

func (pool *TransactionPool) findTransaction(from common.Address, nonce uint64, status byte) *types.Transaction {
	col, ok := pool.accountToTxsMap[from]
	if !ok {
		return nil
	}

	return col.findTx(nonce, status)
}

// GetTransaction returns a transaction if it is contained in the pool and nil otherwise.
func (pool *TransactionPool) GetTransaction(txHash common.Hash) *types.Transaction {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	if pooledTx, ok := pool.hashToTxMap[txHash]; ok {
		return pooledTx.transaction
	}

	return nil
}

// ReflushTransactionStatus reflush the pool transaction status
func (pool *TransactionPool) ReflushTransactionStatus(txHash common.Hash, status byte) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	poolTx := pool.hashToTxMap[txHash]
	if poolTx == nil {
		return
	}
	poolTx.txStatus = status
}

func (pool *TransactionPool) removeTransaction(txHash common.Hash) {
	tx := pool.hashToTxMap[txHash]
	if tx == nil {
		return
	}

	collection := pool.accountToTxsMap[tx.transaction.Data.From]
	if collection != nil {
		collection.remove(tx.transaction.Data.AccountNonce)
		if collection.count(ALL) == 0 {
			delete(pool.accountToTxsMap, tx.transaction.Data.From)
		}
	}

	delete(pool.hashToTxMap, txHash)
}

// RemoveTransactions removes finalized and old transactions in hashToTxMap
func (pool *TransactionPool) RemoveTransactions() {
	for txHash, poolTx := range pool.hashToTxMap {
		txIndex, _ := pool.chain.GetStore().GetTxIndex(txHash)

		state := pool.chain.CurrentState()
		nonce := state.GetNonce(poolTx.transaction.Data.From)

		// Transactions have been processed or are too old need to delete
		if txIndex != nil || poolTx.transaction.Data.AccountNonce+1 < nonce || poolTx.txStatus&ERROR != 0 {
			delete(pool.hashToTxMap, txHash)

			collection := pool.accountToTxsMap[poolTx.transaction.Data.From]
			if collection != nil {
				collection.remove(poolTx.transaction.Data.AccountNonce)
				if collection.count(ALL) == 0 {
					delete(pool.accountToTxsMap, poolTx.transaction.Data.From)
				}
			}
		}
	}
}

// GetProcessableTransactions retrieves all processable transactions. The returned transactions
// are grouped by original account addresses and sorted by nonce ASC.
func (pool *TransactionPool) GetProcessableTransactions() map[common.Address][]*types.Transaction {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	allAccountTxs := make(map[common.Address][]*types.Transaction)

	for account, txs := range pool.accountToTxsMap {
		processableTxs := txs.getTxsOrderByNonceAsc(PENDING)
		if len(processableTxs) != 0 {
			allAccountTxs[account] = processableTxs
		}
	}

	return allAccountTxs
}

// GetProcessableTransactionsCount return the total number of all processable transactions contained within the transaction pool
func (pool *TransactionPool) GetProcessableTransactionsCount() int {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	status := 0
	for _, collection := range pool.accountToTxsMap {
		if collection != nil {
			status += collection.count(PENDING)
		}
	}
	return status
}

// Stop terminates the transaction pool.
func (pool *TransactionPool) Stop() {
}
