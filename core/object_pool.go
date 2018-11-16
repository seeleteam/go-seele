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
	"github.com/seeleteam/go-seele/log"
)

var (
	errObjectHashExists = errors.New("object hash already exists")
	errObjectPoolFull   = errors.New("object pool is full")
	errObjectNonceUsed  = errors.New("object nonce already been used")
)

type blockchain interface {
	GetCurrentState() (*state.Statedb, error)
	GetStore() store.BlockchainStore
}

// poolObject object for pool, like transaction and debt
type poolObject interface {
	FromAccount() common.Address
	Price() *big.Int
	Nonce() uint64
	GetHash() common.Hash
	Size() int
	ToAccount() common.Address
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

type getObjectFromBlockFunc func(block *types.Block) []poolObject
type canRemoveFunc func(chain blockchain, state *state.Statedb, item *poolItem) bool
type objectValidationFunc func(state *state.Statedb, obj poolObject) error
type afterAddFunc func(obj poolObject)

// Pool is a thread-safe container for block object received from the network or submitted locally.
// An object will be removed from the pool once included in a blockchain or pending time too long (> timeoutDuration).
type Pool struct {
	mutex              sync.RWMutex
	capacity           uint
	chain              blockchain
	hashToTxMap        map[common.Hash]*poolItem
	pendingQueue       *pendingQueue
	processingObjects  map[common.Hash]struct{}
	log                *log.SeeleLog
	getObjectFromBlock getObjectFromBlockFunc
	canRemove          canRemoveFunc
	objectValidation   objectValidationFunc
	afterAdd           afterAddFunc
}

// NewPool creates and returns a transaction pool.
func NewPool(capacity uint, chain blockchain, getObjectFromBlock getObjectFromBlockFunc,
	canRemove canRemoveFunc, log *log.SeeleLog, objectValidation objectValidationFunc, afterAdd afterAddFunc) *Pool {
	pool := &Pool{
		capacity:           capacity,
		chain:              chain,
		hashToTxMap:        make(map[common.Hash]*poolItem),
		pendingQueue:       newPendingQueue(),
		processingObjects:  make(map[common.Hash]struct{}),
		log:                log,
		getObjectFromBlock: getObjectFromBlock,
		canRemove:          canRemove,
		objectValidation:   objectValidation,
		afterAdd:           afterAdd,
	}

	return pool
}

// HandleChainHeaderChanged reinjects txs into pool in case of blockchain forked.
func (pool *Pool) HandleChainHeaderChanged(newHeader, lastHeader common.Hash) {
	reinject := pool.getReinjectObject(newHeader, lastHeader)
	count := pool.addObjectArray(reinject)
	if count > 0 {
		pool.log.Info("add %d reinject objects", count)
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
		toDeleted := make(map[common.Hash]poolObject)
		toAdded := make(map[common.Hash]poolObject)
		for newBlock.Header.Height > lastBlock.Header.Height {
			for _, obj := range pool.getObjectFromBlock(newBlock) {
				toDeleted[obj.GetHash()] = obj
			}

			if newBlock, err = chainStore.GetBlock(newBlock.Header.PreviousBlockHash); err != nil {
				log.Error("got block failed, %s", err)
				return nil
			}
		}

		for lastBlock.Header.Height > newBlock.Header.Height {
			for _, obj := range pool.getObjectFromBlock(lastBlock) {
				toAdded[obj.GetHash()] = obj
			}

			if lastBlock, err = chainStore.GetBlock(lastBlock.Header.PreviousBlockHash); err != nil {
				log.Error("got block failed, %s", err)
				return nil
			}
		}

		for lastBlock.HeaderHash != newBlock.HeaderHash {
			for _, obj := range pool.getObjectFromBlock(lastBlock) {
				toAdded[obj.GetHash()] = obj
			}

			for _, obj := range pool.getObjectFromBlock(newBlock) {
				toDeleted[obj.GetHash()] = obj
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

func (pool *Pool) addObjectArray(objects []poolObject) int {
	count := 0
	for _, tx := range objects {
		if err := pool.addObject(tx); err != nil {
			pool.log.Debug("add object failed, %s", err)
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

// addObject adds a single transaction into the pool if it is valid and returns nil.
// Otherwise, return the concrete error.
func (pool *Pool) addObject(obj poolObject) error {
	if pool.Has(obj.GetHash()) {
		return errObjectHashExists
	}

	// validate tx against the latest statedb
	statedb, err := pool.chain.GetCurrentState()
	if err != nil {
		return errors.NewStackedError(err, "failed to get current statedb")
	}

	err = pool.objectValidation(statedb, obj)
	if err != nil {
		return errors.NewStackedError(err, "failed to validate object")
	}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	// avoid to add duplicated obj
	if pool.hashToTxMap[obj.GetHash()] != nil {
		return errObjectHashExists
	}

	// update obj with higher price, otherwise return errObjectNonceUsed
	if existTx := pool.pendingQueue.get(obj.FromAccount(), obj.Nonce()); existTx != nil {
		if obj.Price().Cmp(existTx.Price()) > 0 {
			pool.log.Debug("got a object has higher gas price than before. remove old one. new: %s, old: %s",
				obj.GetHash().Hex(), existTx.GetHash().Hex())
			pool.doRemoveObject(existTx.GetHash())
		} else {
			return errObjectNonceUsed
		}
	}

	// if txpool capacity reached, then discard lower price txs if any.
	// Otherwise, return errObjectPoolFull.
	if uint(len(pool.hashToTxMap)) >= pool.capacity {
		c := pool.pendingQueue.discard(obj.Price())
		if c == nil || c.len() == 0 {
			return errObjectPoolFull
		}

		discardedAccount := c.peek().FromAccount()
		pool.log.Info("object pool is full, discarded account = %v, object len = %v", discardedAccount.Hex(), c.len())

		for c.len() > 0 {
			delete(pool.hashToTxMap, c.pop().GetHash())
		}
	}

	pool.doaddObject(obj)
	pool.afterAdd(obj)

	return nil
}

func (pool *Pool) doaddObject(obj poolObject) {
	poolTx := newPooledItem(obj)
	pool.hashToTxMap[obj.GetHash()] = poolTx
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

// removeOject removes tx of specified tx hash from pool
func (pool *Pool) removeOject(objHash common.Hash) {
	defer pool.mutex.Unlock()
	pool.mutex.Lock()
	pool.doRemoveObject(objHash)
}

// doRemoveObject removes a transaction from pool.
func (pool *Pool) doRemoveObject(objHash common.Hash) {
	if tx := pool.hashToTxMap[objHash]; tx != nil {
		pool.pendingQueue.remove(tx.FromAccount(), tx.Nonce())
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

	for objHash, poolTx := range pool.hashToTxMap {
		if pool.canRemove(pool.chain, state, poolTx) {
			pool.doRemoveObject(objHash)
		}
	}
}

// getProcessableObjects retrieves processable transactions from pool.
func (pool *Pool) getProcessableObjects(size int) ([]poolObject, int) {
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

// getObjectCount return the total number of transactions in the transaction pool.
func (pool *Pool) getObjectCount(processing, pending bool) int {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	count := 0
	if processing {
		count += len(pool.processingObjects)
	}

	if pending {
		count += pool.pendingQueue.count()
	}

	return count
}

// getObjects return the transactions in the transaction pool.
func (pool *Pool) getObjects(processing, pending bool) []poolObject {
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
