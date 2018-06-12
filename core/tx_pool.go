/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"errors"
	"fmt"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
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

const chainHeaderChangeBuffSize = 100

type blockchain interface {
	CurrentState() *state.Statedb
	GetStore() store.BlockchainStore
}

type pooledTx struct {
	*types.Transaction
	txStatus byte
}

// TransactionPool is a thread-safe container for transactions received
// from the network or submitted locally. A transaction will be removed from
// the pool once included in a blockchain.
type TransactionPool struct {
	mutex                    sync.RWMutex
	config                   TransactionPoolConfig
	chain                    blockchain
	hashToTxMap              map[common.Hash]*pooledTx
	accountToTxsMap          map[common.Address]*txCollection // Account address to tx collection mapping.
	lastHeader               common.Hash
	chainHeaderChangeChannel chan common.Hash
	log                      *log.SeeleLog
}

// NewTransactionPool creates and returns a transaction pool.
func NewTransactionPool(config TransactionPoolConfig, chain blockchain) (*TransactionPool, error) {
	header, err := chain.GetStore().GetHeadBlockHash()
	if err != nil {
		return nil, fmt.Errorf("get chain header failed, %s", err)
	}

	pool := &TransactionPool{
		config:          config,
		chain:           chain,
		hashToTxMap:     make(map[common.Hash]*pooledTx),
		accountToTxsMap: make(map[common.Address]*txCollection),
		lastHeader:      header,
		log:             log.GetLogger("txpool", common.LogConfig.PrintLog),
		chainHeaderChangeChannel: make(chan common.Hash, chainHeaderChangeBuffSize),
	}

	event.ChainHeaderChangedEventMananger.AddAsyncListener(pool.chainHeaderChanged)

	return pool, nil
}

// chainHeaderChanged handle chain header changed event.
// add forked transaction back
// deleted invalid transaction
func (pool *TransactionPool) chainHeaderChanged(e event.Event) {
	newHeader := e.(common.Hash)
	pool.chainHeaderChangeChannel <- newHeader
}

// MonitorChainHeaderChange monitor and handle chain header event
func (pool *TransactionPool) MonitorChainHeaderChange() {
	for {
		select {
		case newHeader := <-pool.chainHeaderChangeChannel:
			if pool.lastHeader.IsEmpty() {
				pool.lastHeader = newHeader
				return
			}

			reinject := getReinjectTransaction(pool.chain.GetStore(), newHeader, pool.lastHeader, pool.log)
			pool.addTransactions(reinject)

			pool.lastHeader = newHeader
			pool.RemoveTransactions()
		}
	}
}

func getReinjectTransaction(chainStore store.BlockchainStore, newHeader, lastHeader common.Hash, log *log.SeeleLog) []*types.Transaction {
	newBlock, err := chainStore.GetBlock(newHeader)
	if err != nil {
		log.Error("got block failed, %s", err)
		return nil
	}

	if newBlock.Header.PreviousBlockHash != lastHeader {
		lastBlock, err := chainStore.GetBlock(lastHeader)
		if err != nil {
			log.Error("got block failed, %s", err)
			return nil
		}

		log.Debug("handle chain header forked, last height %d, new height %d", lastBlock.Header.Height, newBlock.Header.Height)
		// add committed txs back in current branch.
		toDeleted := make(map[common.Hash]*types.Transaction)
		toAdded := make(map[common.Hash]*types.Transaction)
		for newBlock.Header.Height > lastBlock.Header.Height {
			for _, t := range newBlock.GetExcludeRewardTransactions() {
				toDeleted[t.Hash] = t
			}

			if newBlock, err = chainStore.GetBlock(newBlock.Header.PreviousBlockHash); err != nil {
				log.Error("got block failed, %s", err)
				return nil
			}
		}

		for lastBlock.Header.Height > newBlock.Header.Height {
			for _, t := range lastBlock.GetExcludeRewardTransactions() {
				toAdded[t.Hash] = t
			}

			if lastBlock, err = chainStore.GetBlock(lastBlock.Header.PreviousBlockHash); err != nil {
				log.Error("got block failed, %s", err)
				return nil
			}
		}

		for lastBlock.HeaderHash != newBlock.HeaderHash {
			for _, t := range lastBlock.GetExcludeRewardTransactions() {
				toAdded[t.Hash] = t
			}

			for _, t := range newBlock.GetExcludeRewardTransactions() {
				toDeleted[t.Hash] = t
			}

			if lastBlock, err = chainStore.GetBlock(lastBlock.Header.PreviousBlockHash); err != nil {
				log.Error("got block failed, %s", err)
				return nil
			}

			if newBlock, err = chainStore.GetBlock(newBlock.Header.PreviousBlockHash); err != nil {
				log.Error("got block failed, %s", err)
				return nil
			}
		}

		reinject := make([]*types.Transaction, 0)
		for key, t := range toAdded {
			if _, ok := toDeleted[key]; !ok {
				reinject = append(reinject, t)
			}
		}

		log.Debug("to added tx length %d, to deleted tx length %d, to reinject tx length %d",
			len(toAdded), len(toDeleted), len(reinject))
		return reinject
	}

	return nil
}

func (pool *TransactionPool) addTransactions(txs []*types.Transaction) {
	if len(txs) == 0 {
		return
	}

	statedb, err := pool.chain.CurrentState().GetCopy()
	if err != nil {
		pool.log.Warn("get stated db failed, %s", err)
		return
	}

	for _, tx := range txs {
		if err := pool.addTransactionWithStateInfo(tx, statedb); err != nil {
			pool.log.Warn("add transaction failed, %s", err)
		}
	}
}

// AddTransaction adds a single transaction into the pool if it is valid and returns nil.
// Otherwise, return the concrete error.
func (pool *TransactionPool) AddTransaction(tx *types.Transaction) error {
	if tx == nil {
		return nil
	}

	statedb, err := pool.chain.CurrentState().GetCopy()
	if err != nil {
		return err
	}

	return pool.addTransactionWithStateInfo(tx, statedb)
}

func (pool *TransactionPool) addTransactionWithStateInfo(tx *types.Transaction, statedb *state.Statedb) error {
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
		return pooledTx.Transaction
	}

	return nil
}

// UpdateTransactionStatus updates the pool transaction status
func (pool *TransactionPool) UpdateTransactionStatus(txHash common.Hash, status byte) {
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

	pool.log.Debug("remove tx hash %s, status %d", txHash.ToHex(), tx.txStatus)

	collection := pool.accountToTxsMap[tx.Data.From]
	if collection != nil {
		collection.remove(tx.Data.AccountNonce)
		if collection.count(ALL) == 0 {
			delete(pool.accountToTxsMap, tx.Data.From)
		}
	}

	delete(pool.hashToTxMap, txHash)
}

// RemoveTransactions removes finalized and old transactions in hashToTxMap
func (pool *TransactionPool) RemoveTransactions() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	for txHash, poolTx := range pool.hashToTxMap {
		txIndex, _ := pool.chain.GetStore().GetTxIndex(txHash)

		state := pool.chain.CurrentState()
		nonce := state.GetNonce(poolTx.Data.From)

		// Transactions have been processed or are too old need to delete
		if txIndex != nil || poolTx.Data.AccountNonce < nonce || poolTx.txStatus&ERROR != 0 {
			pool.log.Debug("remove because of tx already exist %t, nonce too low %t, got error %t, tx nonce %d, target nonce %d",
				txIndex != nil, poolTx.Data.AccountNonce < nonce, poolTx.txStatus&ERROR != 0, poolTx.Data.AccountNonce, nonce)
			pool.removeTransaction(txHash)
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

// GetTransactionsByStatus get transactions by given status
func (pool *TransactionPool) GetTransactionsByStatus(status byte) map[common.Address][]*types.Transaction {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	allAccountTxs := make(map[common.Address][]*types.Transaction)

	for account, txs := range pool.accountToTxsMap {
		processableTxs := txs.getTxsOrderByNonceAsc(status)
		if len(processableTxs) != 0 {
			allAccountTxs[account] = processableTxs
		}
	}

	return allAccountTxs
}

// Stop terminates the transaction pool.
func (pool *TransactionPool) Stop() {
}
