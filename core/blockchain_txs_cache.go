package core

import (
	"math/rand"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

var errTxCacheFull = errors.New("CachedTxs reaches max")
var errDuplicateTx = errors.New("Tx already exists")

const CachedBlocks = uint64(24000)
const PercentDelete = 20 // once CachedTxs reach max, 1/PercentDelete of capacity will be randomly deleted

type CachedTxs struct {
	capacity uint64
	lock     sync.RWMutex
	content  map[common.Hash]*types.Transaction
	log      *log.SeeleLog
}

// 10 * 60 * 60s / 15(s) (block) * 500txs/block = 1.2M txs
// 500 txs = 10KB, so total 1.2M txs will take up 24MB size
func NewCachedTxs(capacity uint64) *CachedTxs {

	return &CachedTxs{
		content:  make(map[common.Hash]*types.Transaction),
		capacity: capacity,
		lock:     sync.RWMutex{},
		log:      log.GetLogger("CachedTxs"),
	}
}

func (c *CachedTxs) init(chain blockchain) error {
	c.log.Info("Initating cached txs within recent %d blocks", CachedBlocks)
	curBlockHash, err := chain.GetStore().GetHeadBlockHash()
	if err != nil {
		c.log.Error("failed to get blockhash form store when cache block txs")
		return err
	}
	curBlock, err := chain.GetStore().GetBlock(curBlockHash)
	if err != nil {
		c.log.Error("failed to get block from store when cache block txs")
		return err
	}
	duplicateTxCount := 0
	txCount := 0
	curHeight := curBlock.Height()
	start := uint64(0)
	if curHeight > CachedBlocks {
		start = curHeight - CachedBlocks
	} else {
		start = 0
	}
	for start < curHeight {
		dup, tc, err := c.getTxsInOneBlock(chain, start)
		if err != nil {
			return err
		}
		duplicateTxCount += dup
		txCount += tc
		start++
	}
	c.log.Warn("[CachedTxs] Cached %d txs, %d existed Txs found", txCount, duplicateTxCount)
	return nil
}

func (c *CachedTxs) getTxsInOneBlock(chain blockchain, h uint64) (int, int, error) {
	// c.log.Info("Getting Txs from %dth Block", h)
	duplicateTxCount := 0
	txCount := 0
	curBlock, err := chain.GetStore().GetBlockByHeight(h)
	if err != nil {
		return 0, duplicateTxCount, err
	}
	txs := curBlock.Transactions
	for i, tx := range txs {
		if i == 0 { // for 1st tx is reward tx, no need to check the duplicate
			continue
		}
		txCount++
		if !c.has(tx.Hash) {
			c.add(tx)
		} else {
			duplicateTxCount++
			c.log.Debug("[CachedTxs] found a duplicate tx %s", tx.Hash)
			continue
		}
	}
	// c.log.Info("%dth Blocks with [%d] txs, [%d] duplicate txs", h, txCount, duplicateTxCount)

	return duplicateTxCount, txCount, nil
}

func (c *CachedTxs) count() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.content)
}

func (c *CachedTxs) add(tx *types.Transaction) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if uint64(len(c.content)) >= c.capacity {
		c.log.Error("Try to randomly remove txs, for %s", errTxCacheFull)
		c.randomDeletes()
		// c.log.Error("after remove, CachedTxs size %d", len(c.content))
	}
	if c.content[tx.Hash] != nil {
		c.log.Debug("Block tx, %s", errDuplicateTx)
	}
	c.content[tx.Hash] = tx
	// fmt.Printf("[CachedTxs] add tx %+v", tx.Hash)
	c.log.Debug("[CachedTxs] add tx %+v", tx.Hash)
}

func (c *CachedTxs) remove(hash common.Hash) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.content, hash)
}

func (c *CachedTxs) has(hash common.Hash) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.content[hash] != nil
}

func (c *CachedTxs) getCachedTxs() []*types.Transaction {
	c.lock.RLock()
	defer c.lock.RUnlock()

	list := make([]*types.Transaction, len(c.content))
	i := 0
	for _, tx := range c.content {
		list[i] = tx
		i++
	}
	return list
}

// randomDeletes randomly delete 1/PercentDelete of map
// random delete make sure the p2p network still have strongest protection from duplicate txs as possible with relasing RAM pressure.
// Furthermore, even in the extreme case : the node own duplicate txs in pool, the node may not the node successfully mined the block.
// At Last, the normal tx, we still have duplicate check in our pool.
func (c *CachedTxs) randomDeletes() {
	rand.Seed(time.Now().UnixNano())
	deleteSize := (int)(len(c.content) / PercentDelete)
	for i := 0; i < deleteSize; i++ {
		k, _ := c.selRand()
		delete(c.content, k)
		i++
	}
}

func (c *CachedTxs) selRand() (k common.Hash, v *types.Transaction) {
	i := rand.Intn(len(c.content)) //24000 * 500
	// since perm or use shuffle will consume either more RAM or more time.
	// Here just use rand.Intn and then iterate the map to delete it.

	for k := range c.content {
		if i == 0 {
			return k, c.content[k]
		}
		i--
	}
	return k, c.content[k]
	panic("never")
}
