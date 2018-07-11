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

// BlockIndex is the index of the block chain
type BlockIndex struct {
	state           *state.Statedb
	currentBlock    *types.Block
	totalDifficulty *big.Int
}

// NewBlockIndex constructs and returns a BlockIndex instance
func NewBlockIndex(state *state.Statedb, block *types.Block, td *big.Int) *BlockIndex {
	return &BlockIndex{
		state:           state,
		currentBlock:    block,
		totalDifficulty: td,
	}
}

// BlockLeaves is the block leaves used for block forking
// Note that BlockLeaves is not thread safe
type BlockLeaves struct {
	blockIndexMap cmap.ConcurrentMap //block hash -> blockIndex

	bestIndex *BlockIndex // the block index which is the first index with the largest total difficulty
}

// NewBlockLeaves constructs and returns a NewBlockLeaves instance
func NewBlockLeaves() *BlockLeaves {
	return &BlockLeaves{
		blockIndexMap: cmap.New(),
	}
}

// Remove removes the specified block index from the block leaves
func (bf *BlockLeaves) Remove(old *BlockIndex) {
	bf.blockIndexMap.Remove(old.currentBlock.HeaderHash.String())
	bf.updateBestIndexWhenRemove(old)
}

// Add adds the specified block index to the block leaves
func (bf *BlockLeaves) Add(index *BlockIndex) {
	bf.blockIndexMap.Set(index.currentBlock.HeaderHash.String(), index)
	bf.updateBestIndexWhenAdd(index)
}

// RemoveByHash removes the block index of the specified hash from the block leaves
func (bf *BlockLeaves) RemoveByHash(hash common.Hash) {
	index := bf.GetBlockIndexByHash(hash)
	bf.blockIndexMap.Remove(hash.String())
	if index != nil {
		bf.updateBestIndexWhenRemove(index)
	}
}

// GetBestBlock gets the current block of the best block index in the block leaves
func (bf *BlockLeaves) GetBestBlock() *types.Block {
	return bf.GetBestBlockIndex().currentBlock
}

// GetBestStateDB gets the state DB of the best block index in the block leaves
func (bf *BlockLeaves) GetBestStateDB() *state.Statedb {
	return bf.GetBestBlockIndex().state
}

// GetBlockIndexByHash gets the block index with the specified hash in the block leaves
func (bf *BlockLeaves) GetBlockIndexByHash(hash common.Hash) *BlockIndex {
	index, ok := bf.blockIndexMap.Get(hash.String())
	if ok {
		return index.(*BlockIndex)
	}

	return nil
}

// Count returns the number of the block indices in the block leaves
func (bf *BlockLeaves) Count() int {
	return bf.blockIndexMap.Count()
}

// GetBestBlockIndex gets the best block index in the block leaves
func (bf *BlockLeaves) GetBestBlockIndex() *BlockIndex {
	return bf.bestIndex
}

// updateBestIndexWhenRemove updates the best index when removing the given block index from the block leaves
func (bf *BlockLeaves) updateBestIndexWhenRemove(index *BlockIndex) {
	if bf.bestIndex != nil && bf.bestIndex.currentBlock.HeaderHash == index.currentBlock.HeaderHash {
		bf.bestIndex = bf.findBestBlockIndex()
	}
}

// updateBestIndexWhenAdd updates the best index when adding the given block index to the block leaves
func (bf *BlockLeaves) updateBestIndexWhenAdd(index *BlockIndex) {
	if bf.bestIndex == nil || bf.bestIndex.totalDifficulty.Cmp(index.totalDifficulty) < 0 {
		bf.bestIndex = index
	}
}

// findBestBlockIndex searchs for the block index of the largest total difficult from the block leaves
func (bf *BlockLeaves) findBestBlockIndex() *BlockIndex {
	maxTD := big.NewInt(0)
	var result *BlockIndex
	for item := range bf.blockIndexMap.IterBuffered() {
		index := item.Val.(*BlockIndex)
		if maxTD.Cmp(index.totalDifficulty) < 0 {
			maxTD = index.totalDifficulty
			result = index
		}
	}

	return result
}

// IsBestBlockIndex indicates whether the given block index is the best compared with all indices in the block leaves
func (bf *BlockLeaves) IsBestBlockIndex(index *BlockIndex) bool {
	td := index.totalDifficulty
	for item := range bf.blockIndexMap.IterBuffered() {
		bi := item.Val.(*BlockIndex)
		if td.Cmp(bi.totalDifficulty) <= 0 {
			return false
		}
	}

	return true
}
