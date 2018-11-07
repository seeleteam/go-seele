/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"container/heap"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/store"
)

const purgeBlockLimit = 500

// BlockIndex is the index of the block chain
type BlockIndex struct {
	blockHash       common.Hash
	blockHeight     uint64
	totalDifficulty *big.Int
}

// NewBlockIndex constructs and returns a BlockIndex instance
func NewBlockIndex(hash common.Hash, height uint64, td *big.Int) *BlockIndex {
	return &BlockIndex{
		blockHash:       hash,
		blockHeight:     height,
		totalDifficulty: td,
	}
}

// cmp compares to the specified block index based on block TD and height.
//   If TD is bigger, return 1.
//   If TD is smaller, return -1.
//   If TD is the same:
//     return 1 for higher height.
//     return 0 for the same height.
//     return -1 for lower height.
func (bi *BlockIndex) cmp(other *BlockIndex) int {
	if r := bi.totalDifficulty.Cmp(other.totalDifficulty); r != 0 {
		return r
	}

	if bi.blockHeight > other.blockHeight {
		return 1
	}

	if bi.blockHeight < other.blockHeight {
		return -1
	}

	return 0
}

type heapedBlockIndex struct {
	common.BaseHeapItem
	*BlockIndex
}

type blockIndices struct {
	bestHeaped  *heapedBlockIndex
	worstHeaped *heapedBlockIndex
}

// BlockLeaves is the block leaves used for block forking
// Note that BlockLeaves is not thread safe
type BlockLeaves struct {
	blockIndexMap map[common.Hash]*blockIndices //block hash -> block indices
	bestHeap      *common.Heap                  // block index heap with best one on top
	worstHeap     *common.Heap                  // block index heap with worst one on top
}

// NewBlockLeaves constructs and returns a NewBlockLeaves instance
func NewBlockLeaves() *BlockLeaves {
	return &BlockLeaves{
		blockIndexMap: make(map[common.Hash]*blockIndices),
		bestHeap: common.NewHeap(func(i, j common.HeapItem) bool {
			iIdx, jIdx := i.(*heapedBlockIndex), j.(*heapedBlockIndex)
			return iIdx.cmp(jIdx.BlockIndex) > 0
		}),
		worstHeap: common.NewHeap(func(i, j common.HeapItem) bool {
			iIdx, jIdx := i.(*heapedBlockIndex), j.(*heapedBlockIndex)
			return iIdx.cmp(jIdx.BlockIndex) < 0
		}),
	}
}

// Add adds the specified block index to the block leaves
func (bf *BlockLeaves) Add(index *BlockIndex) {
	if exist := bf.blockIndexMap[index.blockHash]; exist != nil {
		// update the block index
		exist.bestHeaped.BlockIndex = index
		exist.worstHeaped.BlockIndex = index

		// fix the order in heap
		heap.Fix(bf.bestHeap, exist.bestHeaped.GetHeapIndex())
		heap.Fix(bf.worstHeap, exist.worstHeaped.GetHeapIndex())
	} else {
		indices := &blockIndices{
			bestHeaped:  &heapedBlockIndex{BlockIndex: index},
			worstHeaped: &heapedBlockIndex{BlockIndex: index},
		}

		bf.blockIndexMap[index.blockHash] = indices

		heap.Push(bf.bestHeap, indices.bestHeaped)
		heap.Push(bf.worstHeap, indices.worstHeaped)
	}
}

// Remove removes the block index of the specified hash from the block leaves
func (bf *BlockLeaves) Remove(hash common.Hash) {
	indices := bf.blockIndexMap[hash]
	if indices == nil {
		return
	}

	delete(bf.blockIndexMap, hash)

	heap.Remove(bf.bestHeap, indices.bestHeaped.GetHeapIndex())
	heap.Remove(bf.worstHeap, indices.worstHeaped.GetHeapIndex())
}

// GetBlockIndexByHash gets the block index with the specified hash in the block leaves
func (bf *BlockLeaves) GetBlockIndexByHash(hash common.Hash) *BlockIndex {
	if indices := bf.blockIndexMap[hash]; indices != nil {
		return indices.bestHeaped.BlockIndex
	}

	return nil
}

// Count returns the number of the block indices in the block leaves
func (bf *BlockLeaves) Count() int {
	return len(bf.blockIndexMap)
}

// GetBestBlockIndex gets the best block index in the block leaves
func (bf *BlockLeaves) GetBestBlockIndex() *BlockIndex {
	if best := bf.bestHeap.Peek(); best != nil {
		return best.(*heapedBlockIndex).BlockIndex
	}

	return nil
}

// GetWorstBlockIndex gets the worst block index in the block leaves.
func (bf *BlockLeaves) GetWorstBlockIndex() *BlockIndex {
	if worst := bf.worstHeap.Peek(); worst != nil {
		return worst.(*heapedBlockIndex).BlockIndex
	}

	return nil
}

// IsBestBlockIndex indicates whether the given block index is the best compared with all indices in the block leaves
func (bf *BlockLeaves) IsBestBlockIndex(index *BlockIndex) bool {
	best := bf.GetBestBlockIndex()
	return best == nil || index.cmp(best) > 0
}

// PurgeAsync purges the worst chain in forking tree.
func (bf *BlockLeaves) PurgeAsync(bcStore store.BlockchainStore, callback func(error)) {
	best, worst := bf.GetBestBlockIndex(), bf.GetWorstBlockIndex()
	if best == nil || worst == nil {
		return
	}

	// purge only when worst chain is far away from best chain.
	if best.blockHeight-worst.blockHeight < purgeBlockLimit {
		return
	}

	hash := worst.blockHash
	bf.Remove(hash)

	// asynchronously purge blocks
	go func() {
		err := purgeBlock(hash, bcStore)
		if callback != nil {
			callback(err)
		}
	}()
}

// purgeBlock purges the blocks in forking chain util the common ancestor found in canonical chain.
func purgeBlock(hash common.Hash, bcStore store.BlockchainStore) error {
	for !hash.IsEmpty() {
		header, err := bcStore.GetBlockHeader(hash)
		if err != nil {
			return errors.NewStackedErrorf(err, "failed to get block header by hash %v", hash)
		}

		canonicalHash, err := bcStore.GetBlockHash(header.Height)
		if err != nil {
			return errors.NewStackedErrorf(err, "failed to get block hash by height %v in canonical chain", header.Height)
		}

		// common ancestor found in canonical chain.
		if hash.Equal(canonicalHash) {
			break
		}

		if err := bcStore.DeleteBlock(hash); err != nil {
			return errors.NewStackedErrorf(err, "failed to delete block by hash %v", hash)
		}

		hash = header.PreviousBlockHash
	}

	return nil
}
