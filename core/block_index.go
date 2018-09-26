/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"

	"github.com/orcaman/concurrent-map"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
)

const purgeBlockLimit = uint64(500)

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

// BlockLeaves is the block leaves used for block forking
// Note that BlockLeaves is not thread safe
type BlockLeaves struct {
	blockIndexMap cmap.ConcurrentMap //block hash -> blockIndex

	bestIndex  *BlockIndex // the block index which is the first index with the largest total difficulty
	worstIndex *BlockIndex // the block index which is the first index with the smallest total difficulty
}

// NewBlockLeaves constructs and returns a NewBlockLeaves instance
func NewBlockLeaves() *BlockLeaves {
	return &BlockLeaves{
		blockIndexMap: cmap.New(),
	}
}

// Remove removes the specified block index from the block leaves
func (bf *BlockLeaves) Remove(old *BlockIndex) {
	bf.blockIndexMap.Remove(old.blockHash.String())
	bf.updateIndexWhenRemove(old)
}

// Add adds the specified block index to the block leaves
func (bf *BlockLeaves) Add(index *BlockIndex) {
	bf.blockIndexMap.Set(index.blockHash.String(), index)
	bf.updateIndexWhenAdd(index)
}

// RemoveByHash removes the block index of the specified hash from the block leaves
func (bf *BlockLeaves) RemoveByHash(hash common.Hash) {
	index := bf.GetBlockIndexByHash(hash)
	bf.blockIndexMap.Remove(hash.String())
	if index != nil {
		bf.updateIndexWhenRemove(index)
	}
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

// GetWorstBlockIndex gets the worst block index in the block leaves.
func (bf *BlockLeaves) GetWorstBlockIndex() *BlockIndex {
	return bf.worstIndex
}

// updateIndexWhenRemove updates the best/worst index when removing the given block index from the block leaves
func (bf *BlockLeaves) updateIndexWhenRemove(index *BlockIndex) {
	isBest := bf.bestIndex != nil && bf.bestIndex.blockHash.Equal(index.blockHash)
	isWorst := bf.worstIndex != nil && bf.worstIndex.blockHash.Equal(index.blockHash)
	if isBest || isWorst {
		bf.bestIndex, bf.worstIndex = bf.findBlockIndex()
	}
}

// updateIndexWhenAdd updates the best/worst index when adding the given block index to the block leaves
func (bf *BlockLeaves) updateIndexWhenAdd(index *BlockIndex) {
	if bf.bestIndex == nil || bf.bestIndex.cmp(index) < 0 {
		bf.bestIndex = index
	}

	if bf.worstIndex == nil || bf.worstIndex.cmp(index) > 0 {
		bf.worstIndex = index
	}
}

// findBlockIndex searchs for the block index of the largest and smallest total difficult from the block leaves
func (bf *BlockLeaves) findBlockIndex() (*BlockIndex, *BlockIndex) {
	var best, worst *BlockIndex

	for item := range bf.blockIndexMap.IterBuffered() {
		index := item.Val.(*BlockIndex)

		if best == nil || best.cmp(index) < 0 {
			best = index
		}

		if worst == nil || worst.cmp(index) > 0 {
			worst = index
		}
	}

	return best, worst
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

// Purge purges the worst chain in forking tree.
func (bf *BlockLeaves) Purge(bcStore store.BlockchainStore) error {
	if bf.worstIndex == nil || bf.bestIndex == nil {
		return nil
	}

	// purge only when worst chain is far away from best chain.
	if bf.bestIndex.blockHeight-bf.worstIndex.blockHeight < purgeBlockLimit {
		return nil
	}

	hash := bf.worstIndex.blockHash
	bf.RemoveByHash(hash)

	// remove blocks in worst chain until the common ancestor found in canonical chain.
	for !hash.IsEmpty() {
		header, err := bcStore.GetBlockHeader(hash)
		if err != nil {
			return err
		}

		canonicalHash, err := bcStore.GetBlockHash(header.Height)
		if err != nil {
			return err
		}

		// common ancestor found in canonical chain.
		if hash.Equal(canonicalHash) {
			break
		}

		if err := bcStore.DeleteBlock(hash); err != nil {
			return err
		}

		hash = header.PreviousBlockHash
	}

	return nil
}
