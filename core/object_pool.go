/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
)

var (
	errTxHashExists = errors.New("transaction hash already exists")
	errTxPoolFull   = errors.New("transaction pool is full")
	errTxNonceUsed  = errors.New("transaction nonce already been used")
)

const overTimeInterval = 3 * time.Hour

type blockchain interface {
	GetCurrentState() (*state.Statedb, error)
	GetStore() store.BlockchainStore
}

// poolObject object for pool, like transaction and debt
type poolObject interface {
	Account() common.Address
	Price() *big.Int
	Nonce() uint64
	GetHash() common.Hash
	Size() int
}

// poolItem the item for pool collection
type poolItem struct {
	poolObject
	common.BaseHeapItem
	timestamp time.Time
}

func newPooledItem(object poolObject) *poolItem {
	return &poolItem{
		poolObject: object,
		timestamp:  time.Now(),
	}
}

// Pool is a thread-safe container for transactions received
// from the network or submitted locally. A transaction will be removed from
// the pool once included in a blockchain or pending time too long (> overTimeInterval).
type Pool struct {
	mutex             sync.RWMutex
	capacity          uint
	chain             blockchain
	hashToTxMap       map[common.Hash]*poolItem
	pendingQueue      *pendingQueue
	processingObjects map[common.Hash]struct{}
	log               *log.SeeleLog
}

// NewPool creates and returns a transaction pool.
func NewPool(capacity uint, chain blockchain) *Pool {
	pool := &Pool{
		capacity:          capacity,
		chain:             chain,
		hashToTxMap:       make(map[common.Hash]*poolItem),
		pendingQueue:      newPendingQueue(),
		processingObjects: make(map[common.Hash]struct{}),
		log:               log.GetLogger("txpool"),
	}

	return pool
}

// HandleChainHeaderChanged reinjects txs into pool in case of blockchain forked.
func (pool *Pool) HandleChainHeaderChanged(newHeader, lastHeader common.Hash) {
	reinject := pool.getReinjectObject(newHeader, lastHeader)
	count := pool.addObjects(reinject)
	if count > 0 {
		pool.log.Info("add %d reinject transactions", count)
	}

	pool.removeObjects()
}

func (pool *Pool) getReinjectObject(newHeader, lastHeader common.Hash) []poolObject {
	chainStore := pool.chain.GetStore()
	log := pool.log

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

		reinject := make([]poolObject, 0)
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

func (pool *Pool) addObjects(txs []poolObject) int {
	count := 0
	for _, tx := range txs {
		if err := pool.AddObject(tx); err != nil {
			pool.log.Debug("add transaction failed, %s", err)
		} else {
			count++
		}
	}

	return count
}

func (pool *Pool) Has(hash common.Hash) bool {
	// return immediately if tx already exists
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	return pool.hashToTxMap[hash] != nil
}

// AddObject adds a single transaction into the pool if it is valid and returns nil.
// Otherwise, return the concrete error.
func (pool *Pool) AddObject(obj poolObject) error {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	// avoid to add duplicated obj
	if pool.hashToTxMap[obj.GetHash()] != nil {
		return errTxHashExists
	}

	// update obj with higher price, otherwise return errTxNonceUsed
	if existTx := pool.pendingQueue.get(obj.Account(), obj.Nonce()); existTx != nil {
		if obj.Price().Cmp(existTx.Price()) > 0 {
			pool.log.Debug("got a transaction have higher gas price than before. remove old one. new: %s, old: %s",
				obj.GetHash().ToHex(), existTx.GetHash().ToHex())
			pool.doRemoveObject(existTx.GetHash())
		} else {
			return errTxNonceUsed
		}
	}

	// if txpool capacity reached, then discard lower price txs if any.
	// Otherwise, return errTxPoolFull.
	if uint(len(pool.hashToTxMap)) >= pool.capacity {
		c := pool.pendingQueue.discard(obj.Price())
		if c == nil || c.len() == 0 {
			return errTxPoolFull
		}

		discardedAccount := c.peek().Account()
		pool.log.Info("txpool is full, discarded account = %v, txs = %v", discardedAccount.ToHex(), c.len())

		for c.len() > 0 {
			delete(pool.hashToTxMap, c.pop().GetHash())
		}
	}

	pool.addObject(obj)
	pool.log.Debug("receive transaction and add it. transaction hash: %v, time: %d", obj.GetHash(), time.Now().UnixNano())
	// fire event
	event.TransactionInsertedEventManager.Fire(obj)

	return nil
}

func (pool *Pool) addObject(tx poolObject) {
	poolTx := newPooledItem(tx)
	pool.hashToTxMap[tx.GetHash()] = poolTx
	pool.pendingQueue.add(poolTx)
}

// GetObject returns a transaction if it is contained in the pool and nil otherwise.
func (pool *Pool) GetObject(objHash common.Hash) poolObject {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	if pooledTx, ok := pool.hashToTxMap[objHash]; ok {
		return pooledTx.poolObject
	}

	return nil
}

// RemoveOject removes tx of specified tx hash from pool
func (pool *Pool) RemoveOject(objHash common.Hash) {
	defer pool.mutex.Unlock()
	pool.mutex.Lock()
	pool.doRemoveObject(objHash)
}

// doRemoveObject removes a transaction from pool.
func (pool *Pool) doRemoveObject(objHash common.Hash) {
	if tx := pool.hashToTxMap[objHash]; tx != nil {
		pool.pendingQueue.remove(tx.Account(), tx.Nonce())
		delete(pool.processingObjects, objHash)
		delete(pool.hashToTxMap, objHash)
	}
}

// removeObjects removes finalized and old transactions in hashToTxMap
func (pool *Pool) removeObjects() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	state, err := pool.chain.GetCurrentState()
	if err != nil {
		pool.log.Warn("failed to get current state, err: %s", err)
		return
	}

	nowTimestamp := time.Now()
	for objHash, poolTx := range pool.hashToTxMap {
		txIndex, _ := pool.chain.GetStore().GetTxIndex(objHash)
		nonce := state.GetNonce(poolTx.Account())
		duration := nowTimestamp.Sub(poolTx.timestamp)

		// Transactions have been processed or are too old need to delete
		if txIndex != nil || poolTx.Nonce() < nonce || duration > overTimeInterval {
			if txIndex == nil {
				if poolTx.Nonce() < nonce {
					pool.log.Debug("remove tx %s because nonce too low, account %s, tx nonce %d, target nonce %d", objHash.ToHex(),
						poolTx.Account().ToHex(), poolTx.Nonce(), nonce)
				} else if duration > overTimeInterval {
					pool.log.Debug("remove tx %s because not packed for more than three hours", objHash.ToHex())
				}
			}
			pool.doRemoveObject(objHash)
		}
	}
}

// GetProcessableObjects retrieves processable transactions from pool.
func (pool *Pool) GetProcessableObjects(size int) ([]poolObject, int) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	totalSize := 0
	var txs []poolObject

	for !pool.pendingQueue.empty() {
		tx := pool.pendingQueue.peek().peek().poolObject
		tmpSize := totalSize + tx.Size()
		if tmpSize > size {
			break
		}

		tx = pool.pendingQueue.pop()
		totalSize = tmpSize
		txs = append(txs, tx)
		pool.processingObjects[tx.GetHash()] = struct{}{}
	}

	return txs, totalSize
}

// GetPendingObjectCount return the total number of pending transactions in the transaction pool.
func (pool *Pool) GetPendingObjectCount() int {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	return pool.pendingQueue.count()
}

// GetObjectCount return the total number of transactions in the transaction pool.
func (pool *Pool) GetObjectCount() int {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	return pool.pendingQueue.count() + len(pool.processingObjects)
}

// GetObjects return the transactions in the transaction pool.
func (pool *Pool) GetObjects(processing, pending bool) []poolObject {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	var txs []poolObject

	if processing {
		for hash := range pool.processingObjects {
			if tx := pool.hashToTxMap[hash]; tx != nil {
				txs = append(txs, tx.poolObject)
			}
		}
	}

	if pending {
		pendingTxs := pool.pendingQueue.list()
		txs = append(txs, pendingTxs...)
	}

	return txs
}
