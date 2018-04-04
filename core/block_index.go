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

// BlockLeaf block leafs used for block fork
type BlockLeaf struct {
	blockIndexMap cmap.ConcurrentMap //block hash -> blockIndex
}

func NewBlockLeaf() *BlockLeaf {
	return &BlockLeaf{
		blockIndexMap: cmap.New(),
	}
}

func (bf *BlockLeaf) Remove(old *BlockIndex) {
	bf.blockIndexMap.Remove(old.currentBlock.HeaderHash.String())
}

func (bf *BlockLeaf) Add(index *BlockIndex) {
	bf.blockIndexMap.Set(index.currentBlock.HeaderHash.String(), index)
}

func (bf *BlockLeaf) RemoveByHash(hash common.Hash) {
	bf.blockIndexMap.Remove(hash.String())
}

func (bf *BlockLeaf) GetBestBlock() *types.Block {
	return bf.GetBestBlockIndex().currentBlock
}

func (bf *BlockLeaf) GetBestStateDB() *state.Statedb {
	return bf.GetBestBlockIndex().state
}

func (bf *BlockLeaf) GetBlockIndexByHash(hash common.Hash) *BlockIndex {
	index, ok := bf.blockIndexMap.Get(hash.String())
	if ok {
		return index.(*BlockIndex)
	}

	return nil
}

func (bf *BlockLeaf) Count() int {
	return bf.blockIndexMap.Count()
}

func (bf *BlockLeaf) GetBestBlockIndex() *BlockIndex {
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

func (bf *BlockLeaf) IsBestBlockIndex(index *BlockIndex) bool {
	td := index.totalDifficult
	for item := range bf.blockIndexMap.IterBuffered() {
		bi := item.Val.(*BlockIndex)
		if td.Cmp(bi.totalDifficult) <= 0 {
			return false
		}
	}

	return true
}
