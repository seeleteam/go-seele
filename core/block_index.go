/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"

	"github.com/orcaman/concurrent-map"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
)

// BlockIndex index of the block chain
type BlockIndex struct {
	state          *state.Statedb
	currentBlock   *types.Block
	totalDifficult *big.Int
}

func NewBlockIndex(state *state.Statedb, block *types.Block, td *big.Int) *BlockIndex {
	return &BlockIndex{
		state:          state,
		currentBlock:   block,
		totalDifficult: td,
	}
}

// BlockLeaves block leafs used for block fork
// Note BlockLeaves is not thread safe
type BlockLeaves struct {
	blockIndexMap cmap.ConcurrentMap //block hash -> blockIndex

	bestIndex *BlockIndex // the first and largest total difficult block index
}

func NewBlockLeaf() *BlockLeaves {
	return &BlockLeaves{
		blockIndexMap: cmap.New(),
	}
}

func (bf *BlockLeaves) Remove(old *BlockIndex) {
	bf.blockIndexMap.Remove(old.currentBlock.HeaderHash.String())
	bf.updateBestIndexWhenRemove(old)
}

func (bf *BlockLeaves) Add(index *BlockIndex) {
	bf.blockIndexMap.Set(index.currentBlock.HeaderHash.String(), index)
	bf.updateBestIndexWhenAdd(index)
}

func (bf *BlockLeaves) RemoveByHash(hash common.Hash) {
	index := bf.GetBlockIndexByHash(hash)
	bf.blockIndexMap.Remove(hash.String())
	if index != nil {
		bf.updateBestIndexWhenRemove(index)
	}
}

func (bf *BlockLeaves) GetBestBlock() *types.Block {
	return bf.GetBestBlockIndex().currentBlock
}

func (bf *BlockLeaves) GetBestStateDB() *state.Statedb {
	return bf.GetBestBlockIndex().state
}

func (bf *BlockLeaves) GetBlockIndexByHash(hash common.Hash) *BlockIndex {
	index, ok := bf.blockIndexMap.Get(hash.String())
	if ok {
		return index.(*BlockIndex)
	}

	return nil
}

func (bf *BlockLeaves) Count() int {
	return bf.blockIndexMap.Count()
}

func (bf *BlockLeaves) GetBestBlockIndex() *BlockIndex {
	return bf.bestIndex
}

func (bf *BlockLeaves) updateBestIndexWhenRemove(index *BlockIndex) {
	if bf.bestIndex != nil && bf.bestIndex.currentBlock.HeaderHash == index.currentBlock.HeaderHash {
		bf.bestIndex = bf.findBestBlockHash()
	}
}

func (bf *BlockLeaves) updateBestIndexWhenAdd(index *BlockIndex) {
	if bf.bestIndex == nil || bf.bestIndex.totalDifficult.Cmp(index.totalDifficult) < 0 {
		bf.bestIndex = index
	}
}

func (bf *BlockLeaves) findBestBlockHash() *BlockIndex {
	maxTD := big.NewInt(0)
	var result *BlockIndex
	for item := range bf.blockIndexMap.IterBuffered() {
		index := item.Val.(*BlockIndex)
		if maxTD.Cmp(index.totalDifficult) < 0 {
			maxTD = index.totalDifficult
			result = index
		}
	}

	return result
}

func (bf *BlockLeaves) IsBestBlockIndex(index *BlockIndex) bool {
	td := index.totalDifficult
	for item := range bf.blockIndexMap.IterBuffered() {
		bi := item.Val.(*BlockIndex)
		if td.Cmp(bi.totalDifficult) <= 0 {
			return false
		}
	}

	return true
}
