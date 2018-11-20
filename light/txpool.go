/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"fmt"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
)

const (
	headerChanBufSize = 16
	txConfirmBlocks   = uint64(500)
)

type minedBlock struct {
	height uint64
	txs    []*types.Transaction
}

type txPool struct {
	mutex                     sync.RWMutex
	chain                     BlockChain
	odrBackend                *odrBackend
	pendingTxs                map[common.Hash]*types.Transaction // txs that not mined yet.
	minedBlocks               map[common.Hash]*minedBlock        // blocks that already mined.
	packTxs                   map[common.Hash]common.Hash        // Txs that already packed.
	headerCh                  chan *types.BlockHeader            // channel to receive new header in canonical chain.
	currentHeader             *types.BlockHeader                 // current HEAD header in canonical chain.
	headerChangedEventManager *event.EventManager
	log                       *log.SeeleLog
}

func newTxPool(chain BlockChain, odrBackend *odrBackend, headerChangedEventManager *event.EventManager) *txPool {
	pool := &txPool{
		chain:                     chain,
		odrBackend:                odrBackend,
		pendingTxs:                make(map[common.Hash]*types.Transaction),
		minedBlocks:               make(map[common.Hash]*minedBlock),
		packTxs:                   make(map[common.Hash]common.Hash),
		headerCh:                  make(chan *types.BlockHeader, headerChanBufSize),
		currentHeader:             chain.CurrentHeader(),
		headerChangedEventManager: headerChangedEventManager,
		log: log.GetLogger("lightTxPool"),
	}

	headerChangedEventManager.AddAsyncListener(pool.onBlockHeaderChanged)

	go pool.eventLoop()

	return pool
}

// AddTransaction sends the specified tx to remote peer via odr backend.
func (pool *txPool) AddTransaction(tx *types.Transaction) error {
	if tx == nil {
		return nil
	}

	if err := tx.ValidateWithoutState(true, false); err != nil {
		return errors.NewStackedError(err, "failed to validate tx without state")
	}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if pool.pendingTxs[tx.Hash] != nil {
		return fmt.Errorf("Transaction already exists, hash is %v", tx.Hash.Hex())
	}

	if _, err := pool.odrBackend.retrieve(&odrAddTx{Tx: *tx}); err != nil {
		return errors.NewStackedError(err, "failed to add tx via ODR backend")
	}

	pool.pendingTxs[tx.Hash] = tx

	return nil
}

// GetTransaction returns a transaction if it is contained in the pool and nil otherwise.
func (pool *txPool) GetTransaction(txHash common.Hash) *types.Transaction {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	return pool.pendingTxs[txHash]
}

// GetTransactions return the transactions in the transaction pool.
func (pool *txPool) GetTransactions(processing, pending bool) []*types.Transaction {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	if !pending || len(pool.pendingTxs) == 0 {
		return nil
	}

	txs := make([]*types.Transaction, len(pool.pendingTxs))
	i := 0

	for _, tx := range pool.pendingTxs {
		txs[i] = tx
		i++
	}

	return txs
}

// Remove removes tx of specified tx hash from pool.
func (pool *txPool) Remove(txHash common.Hash) {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	delete(pool.pendingTxs, txHash)
}

// GetPendingTxCount return the total number of pending transactions in the transaction pool.
func (pool *txPool) GetPendingTxCount() int {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()

	return len(pool.pendingTxs)
}

// GetTxCount return the total number of transactions in the transaction pool.
func (pool *txPool) GetTxCount() int {
	return pool.GetPendingTxCount()
}

func (pool *txPool) stop() {
	pool.headerChangedEventManager.RemoveListener(pool.onBlockHeaderChanged)
	close(pool.headerCh)
}

func (pool *txPool) onBlockHeaderChanged(e event.Event) {
	pool.headerCh <- e.(*types.BlockHeader)
}

func (pool *txPool) eventLoop() {
	for {
		select {
		case newHeader := <-pool.headerCh:
			if err := pool.setNewHeader(newHeader); err != nil {
				pool.log.Error(errors.NewStackedError(err, "failed to set new header").Error())
			}
		}
	}
}

func (pool *txPool) setNewHeader(newHeader *types.BlockHeader) error {
	if newHeader == nil {
		return nil
	}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	oldHeader := pool.currentHeader
	pool.currentHeader = newHeader

	oldHashes, newHashes, err := pool.commonAncestor(oldHeader, newHeader)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to find common ancestor, old = %+v, new = %+v", oldHeader, newHeader)
	}

	for _, blockHash := range oldHashes {
		pool.rollbackTxs(blockHash)
	}

	for _, blockHash := range newHashes {
		if err := pool.checkMinedTxs(blockHash); err != nil {
			return errors.NewStackedErrorf(err, "failed to check mined txs for block %v", blockHash)
		}
	}

	pool.clearConfirmedBlocks()

	return nil
}

// commonAncestor find the common ancestor for the specified old and new block headers.
// It returns the old and new block hashes in canonical chain.
func (pool *txPool) commonAncestor(oldHeader, newHeader *types.BlockHeader) (oldHashes, newHashes []common.Hash, err error) {
	oldHash, newHash := oldHeader.Hash(), newHeader.Hash()
	var preHash common.Hash

	for !oldHash.Equal(newHash) {
		if oldHeader.Height >= newHeader.Height {
			// old canonical chain
			oldHashes = append(oldHashes, oldHash)
			preHash = oldHeader.PreviousBlockHash
			if oldHeader, err = pool.chain.GetStore().GetBlockHeader(preHash); err != nil {
				return nil, nil, errors.NewStackedErrorf(err, "failed to get block header by hash %v", preHash)
			}
			oldHash = preHash
		} else {
			// new canonical chain
			newHashes = append(newHashes, newHash)
			preHash = newHeader.PreviousBlockHash
			if newHeader, err = pool.chain.GetStore().GetBlockHeader(preHash); err != nil {
				return nil, nil, errors.NewStackedErrorf(err, "failed to get block header by hash %v", preHash)
			}
			newHash = preHash
		}
	}

	return oldHashes, newHashes, nil
}

// rollbackTxs roll back txs of the specified block hash back into tx pool.
func (pool *txPool) rollbackTxs(blockHash common.Hash) {
	block := pool.minedBlocks[blockHash]
	if block == nil {
		return
	}

	for _, tx := range block.txs {
		pool.pendingTxs[tx.Hash] = tx
		delete(pool.packTxs, tx.Hash)
	}

	delete(pool.minedBlocks, blockHash)
}

// checkMinedTxs retrieves block of the specified block hash via odr backend,
// and update the txs status from pending to mined.
func (pool *txPool) checkMinedTxs(blockHash common.Hash) error {
	// do nothing if no pending txs.
	if len(pool.pendingTxs) == 0 {
		return nil
	}

	block, err := pool.getBlock(blockHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block from remote peer, hash = %v", blockHash)
	}

	var minedTxs []*types.Transaction
	for _, tx := range block.Transactions {
		if _, ok := pool.pendingTxs[tx.Hash]; ok {
			minedTxs = append(minedTxs, tx)
		}
	}

	if minedTxs != nil {
		pool.minedBlocks[blockHash] = &minedBlock{
			height: block.Header.Height,
			txs:    minedTxs,
		}

		for _, tx := range minedTxs {
			pool.packTxs[tx.Hash] = block.HeaderHash
			delete(pool.pendingTxs, tx.Hash)
		}
	}

	return nil
}

// getBlock retrieves block of specified block hash via odr backend.
func (pool *txPool) getBlock(hash common.Hash) (*types.Block, error) {
	request := &odrBlock{Hash: hash}
	filter := peerFilter{blockHash: hash}
	var counter int

	// allow to try 3 times
	for {
		response, err := pool.odrBackend.retrieveWithFilter(request, filter)
		if err == nil {
			return response.(*odrBlock).Block, nil
		}

		pool.log.Info(errors.NewStackedErrorf(err, "failed to retrieve ODR block, counter = %v", counter).Error())
		counter++

		if counter == 3 {
			return nil, errors.NewStackedError(err, "failed to retrieve ODR block")
		}
	}
}

// clearConfirmedBlocks clears the confirmed txs from tx pool.
func (pool *txPool) clearConfirmedBlocks() {
	confirmedHeight := pool.currentHeader.Height - txConfirmBlocks
	if confirmedHeight <= 0 {
		return
	}

	var confirmedBlocks []common.Hash

	for hash, block := range pool.minedBlocks {
		if block.height <= confirmedHeight {
			confirmedBlocks = append(confirmedBlocks, hash)
		}
	}

	for _, hash := range confirmedBlocks {
		for i := 0; i < len(pool.minedBlocks[hash].txs); i++ {
			delete(pool.packTxs, pool.minedBlocks[hash].txs[i].Hash)
		}
		delete(pool.minedBlocks, hash)
	}
}

func (pool *txPool) GetBlockHash(txHash common.Hash) common.Hash {
	defer pool.mutex.RUnlock()
	pool.mutex.RLock()
	if blockHash, ok := pool.packTxs[txHash]; ok {
		return blockHash
	}
	return common.EmptyHash
}
