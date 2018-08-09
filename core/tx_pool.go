/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"errors"
	"fmt"
	"sync"
	"time"

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
	errTxNonceUsed  = errors.New("transaction nonce already been used")
)

const chainHeaderChangeBuffSize = 100
const overTimeInterval = 3 * time.Hour

type blockchain interface {
	GetCurrentState() (*state.Statedb, error)
	GetStore() store.BlockchainStore
}

type pooledTx struct {
	*types.Transaction
	heapItem
	timestamp time.Time
}

func newPooledTx(tx *types.Transaction) *pooledTx {
	return &pooledTx{tx, heapItem{0}, time.Now()}
}

// TransactionPool is a thread-safe container for transactions received
// from the network or submitted locally. A transaction will be removed from
// the pool once included in a blockchain or pending time too long (> overTimeInterval).
type TransactionPool struct {
	mutex                    sync.RWMutex
	config                   TransactionPoolConfig
	chain                    blockchain
	hashToTxMap              map[common.Hash]*pooledTx
	pendingQueue             *pendingQueue
	processingTxs            map[common.Hash]struct{}
	lastHeader               common.Hash
	chainHeaderChangeChannel chan common.Hash
	log                      *log.SeeleLog
}

// NewTransactionPool creates and returns a transaction pool.
func NewTransactionPool(config TransactionPoolConfig, chain blockchain) (*TransactionPool, error) {
	header, err := chain.GetStore().GetHeadBlockHash()
	if err != nil {
		return nil, fmt.Errorf("failed to get chain header, %s", err)
	}

	pool := &TransactionPool{
		config:        config,
		chain:         chain,
		hashToTxMap:   make(map[common.Hash]*pooledTx),
		pendingQueue:  newPendingQueue(),
		processingTxs: make(map[common.Hash]struct{}),
		lastHeader:    header,
		log:           log.GetLogger("txpool"),
		chainHeaderChangeChannel: make(chan common.Hash, chainHeaderChangeBuffSize),
	}

	event.ChainHeaderChangedEventMananger.AddAsyncListener(pool.chainHeaderChanged)
	go pool.MonitorChainHeaderChange()

	return pool, nil
}

// chainHeaderChanged handle chain header changed event.
// add forked transaction back
// deleted invalid transaction
func (pool *TransactionPool) chainHeaderChanged(e event.Event) {
	newHeader := e.(common.Hash)
	if newHeader.IsEmpty() {
		return
	}

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
			count := pool.addTransactions(reinject)
			if count > 0 {
				pool.log.Info("add %d reinject transactions", count)
			}

			pool.lastHeader = newHeader
			pool.removeTransactions()
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

func (pool *TransactionPool) addTransactions(txs []*types.Transaction) int {
	count := 0
	for _, tx := range txs {
		if err := pool.AddTransaction(tx); err != nil {
			pool.log.Debug("add transaction failed, %s", err)
		} else {
			count++
		}
	}

	return count
}

// AddTransaction adds a single transaction into the pool if it is valid and returns nil.
// Otherwise, return the concrete error.
func (pool *TransactionPool) AddTransaction(tx *types.Transaction) error {
	if tx == nil {
		return nil
	}

	statedb, err := pool.chain.GetCurrentState()
	if err != nil {
		return fmt.Errorf("get current state db failed, error %s", err)
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

	if existTx := pool.pendingQueue.get(tx.Data.From, tx.Data.AccountNonce); existTx != nil {
		if tx.Data.Fee.Cmp(existTx.Data.Fee) > 0 {
			pool.log.Debug("got a transaction have more fees than before. remove old one. new: %s, old: %s",
				tx.Hash.ToHex(), existTx.Hash.ToHex())
			pool.RemoveTransaction(existTx.Hash)
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
	poolTx := newPooledTx(tx)
	pool.hashToTxMap[tx.Hash] = poolTx
	pool.pendingQueue.add(poolTx)
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

// RemoveTransaction removes a transaction from pool.
func (pool *TransactionPool) RemoveTransaction(txHash common.Hash) {
	if tx := pool.hashToTxMap[txHash]; tx != nil {
		pool.pendingQueue.remove(tx.Data.From, tx.Data.AccountNonce)
		delete(pool.processingTxs, txHash)
		delete(pool.hashToTxMap, txHash)
	}
}

// removeTransactions removes finalized and old transactions in hashToTxMap
func (pool *TransactionPool) removeTransactions() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	state, err := pool.chain.GetCurrentState()
	if err != nil {
		pool.log.Warn("failed to get current state, err: %s", err)
		return
	}

	nowTimestamp := time.Now()
	for txHash, poolTx := range pool.hashToTxMap {
		txIndex, _ := pool.chain.GetStore().GetTxIndex(txHash)
		nonce := state.GetNonce(poolTx.Data.From)
		duration := nowTimestamp.Sub(poolTx.timestamp)

		// Transactions have been processed or are too old need to delete
		if txIndex != nil || poolTx.Data.AccountNonce < nonce || duration > overTimeInterval {
			if txIndex == nil {
				if poolTx.Data.AccountNonce < nonce {
					pool.log.Debug("remove tx %s because nonce too low, account %s, tx nonce %d, target nonce %d", txHash.ToHex(),
						poolTx.Data.From.ToHex(), poolTx.Data.AccountNonce, nonce)
				} else if duration > overTimeInterval {
					pool.log.Debug("remove tx %s because not packed for more than three hours", txHash.ToHex())
				}
			}
			pool.RemoveTransaction(txHash)
		}
	}
}

// GetProcessableTransactions retrieves processable transactions from pool.
func (pool *TransactionPool) GetProcessableTransactions(size int) ([]*types.Transaction, int) {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	totalSize := 0
	var txs []*types.Transaction

	for pool.pendingQueue.feeHeap.Len() > 0 {
		tx := pool.pendingQueue.peek().peek().Transaction
		tmpSize := totalSize + tx.Size()
		if tmpSize > size {
			break
		}

		tx = pool.pendingQueue.pop()
		totalSize = tmpSize
		txs = append(txs, tx)
		pool.processingTxs[tx.Hash] = struct{}{}
	}

	return txs, totalSize
}

// GetPendingTxCount return the total number of pending transactions in the transaction pool.
func (pool *TransactionPool) GetPendingTxCount() int {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	return pool.pendingQueue.count()
}

// GetTransactions return the transactions in the transaction pool.
func (pool *TransactionPool) GetTransactions(processing, pending bool) []*types.Transaction {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	var txs []*types.Transaction

	if processing {
		for hash := range pool.processingTxs {
			if tx := pool.hashToTxMap[hash]; tx != nil {
				txs = append(txs, tx.Transaction)
			}
		}
	}

	if pending {
		pendingTxs := pool.pendingQueue.list()
		txs = append(txs, pendingTxs...)
	}

	return txs
}

// Stop terminates the transaction pool.
func (pool *TransactionPool) Stop() {
}
