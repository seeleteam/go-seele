/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/core/store"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/stretchr/testify/assert"
)

func newTestBlockIndex(hash string, td int64, height uint64) *BlockIndex {
	return NewBlockIndex(common.StringToHash(hash), height, big.NewInt(td))
}

func Test_BlockIndex_cmp(t *testing.T) {
	idx := newTestBlockIndex("b1", 2, 2)

	// compare with samller TD
	assert.Equal(t, 1, idx.cmp(newTestBlockIndex("b2", 1, 1))) // samller height
	assert.Equal(t, 1, idx.cmp(newTestBlockIndex("b2", 1, 2))) // same height
	assert.Equal(t, 1, idx.cmp(newTestBlockIndex("b2", 1, 3))) // bigger height

	// compare with same TD
	assert.Equal(t, 1, idx.cmp(newTestBlockIndex("b2", 2, 1)))  // samller height
	assert.Equal(t, 0, idx.cmp(newTestBlockIndex("b2", 2, 2)))  // same height
	assert.Equal(t, -1, idx.cmp(newTestBlockIndex("b2", 2, 3))) // bigger height

	// compare with bigger TD
	assert.Equal(t, -1, idx.cmp(newTestBlockIndex("b2", 3, 1))) // samller height
	assert.Equal(t, -1, idx.cmp(newTestBlockIndex("b2", 3, 2))) // same height
	assert.Equal(t, -1, idx.cmp(newTestBlockIndex("b2", 3, 3))) // bigger height
}

func Test_BlockLeaf_Add_Remove(t *testing.T) {
	bf := NewBlockLeaves()

	index := newTestBlockIndex("block 1", 1, 1)
	bf.Add(index)
	assert.Equal(t, bf.Count(), 1)

	index2 := newTestBlockIndex("block 2", 2, 2)
	bf.Add(index2)
	assert.Equal(t, bf.Count(), 2)

	bf.Remove(index.blockHash)
	assert.Equal(t, bf.Count(), 1)

	bf.Remove(index2.blockHash)
	assert.Equal(t, bf.Count(), 0)
}

func Test_BlockLeaf_Get(t *testing.T) {
	bf := NewBlockLeaves()
	index := newTestBlockIndex("block 1", 1, 1)
	bf.Add(index)
	index2 := newTestBlockIndex("block 2", 2, 2)
	bf.Add(index2)

	assert.Equal(t, bf.GetBestBlockIndex(), index2)
	assert.Equal(t, bf.GetWorstBlockIndex(), index)

	assert.Equal(t, bf.GetBlockIndexByHash(index.blockHash), index)

	index3 := newTestBlockIndex("block 3", 2, 1)
	assert.Equal(t, bf.IsBestBlockIndex(index3), false)

	index4 := newTestBlockIndex("block 4", 3, 4)
	assert.Equal(t, bf.IsBestBlockIndex(index4), true)
}

func Test_BlockLeaf_Purge_NoAction(t *testing.T) {
	bf := NewBlockLeaves()

	// purge on empty case
	bf.PurgeAsync(nil, func(err error) {
		assert.Nil(t, err)
	})

	bf.Add(newTestBlockIndex("b1", 1000, 1000))
	bf.Add(newTestBlockIndex("b2", 501, 501))

	// do nothing purgeBlockLimit not reached.
	bf.PurgeAsync(nil, func(err error) {
		assert.Nil(t, err)
		assert.Equal(t, bf.Count(), 2)
	})
}

func Test_BlockLeaf_Purge(t *testing.T) {
	blockFactory := func(preBlockHash common.Hash, height uint64, createTime int) *types.Block {
		header := &types.BlockHeader{
			PreviousBlockHash: preBlockHash,
			Difficulty:        big.NewInt(1),
			Height:            height,
			CreateTimestamp:   big.NewInt(int64(createTime)),
		}

		return &types.Block{
			HeaderHash: header.Hash(),
			Header:     header,
		}
	}

	bf := NewBlockLeaves()
	bcStore := store.NewMemStore()

	ancestor := blockFactory(common.StringToHash("ancestor block"), 38, 1)
	td := new(big.Int).SetUint64(ancestor.Header.Height)
	bcStore.PutBlock(ancestor, td, true)

	numForkingBlocks := 3

	// construct canonical chain
	canonicalBlock := ancestor
	for i := 0; i < purgeBlockLimit+numForkingBlocks; i++ {
		canonicalBlock = blockFactory(canonicalBlock.HeaderHash, canonicalBlock.Header.Height+1, 1)
		td = new(big.Int).SetUint64(canonicalBlock.Header.Height)
		assert.Nil(t, bcStore.PutBlock(canonicalBlock, td, true))
	}
	bf.Add(NewBlockIndex(canonicalBlock.HeaderHash, canonicalBlock.Header.Height, td))

	// construct forking chain
	forkingBlocks := []*types.Block{ancestor}
	for i := 0; i < numForkingBlocks; i++ {
		preBlock, newHeight := forkingBlocks[i], forkingBlocks[i].Header.Height+1
		block := blockFactory(preBlock.HeaderHash, newHeight, 2)
		td = new(big.Int).SetUint64(newHeight)
		assert.Nil(t, bcStore.PutBlock(block, td, false))
		forkingBlocks = append(forkingBlocks, block)
	}
	forkingBlock := forkingBlocks[numForkingBlocks]
	bf.Add(NewBlockIndex(forkingBlock.HeaderHash, forkingBlock.Header.Height, td))

	// check states before purge
	assert.Equal(t, 2, bf.Count())
	assert.Equal(t, canonicalBlock.HeaderHash, bf.GetBestBlockIndex().blockHash)
	assert.Equal(t, forkingBlock.HeaderHash, bf.GetWorstBlockIndex().blockHash)

	// purge once
	bf.PurgeAsync(bcStore, func(err error) {
		assert.Nil(t, err)

		// check states after purge
		assert.Equal(t, 1, bf.Count())
		assert.Equal(t, canonicalBlock.HeaderHash, bf.GetBestBlockIndex().blockHash)
		assert.Equal(t, canonicalBlock.HeaderHash, bf.GetWorstBlockIndex().blockHash)

		// check the blockchain store to ensure forking blocks purged.
		for _, block := range forkingBlocks {
			exists, err := bcStore.HasBlock(block.HeaderHash)
			assert.Nil(t, err)
			// all forking blocks purged except the ancestor.
			assert.Equal(t, block.HeaderHash.Equal(ancestor.HeaderHash), exists)
		}
	})
}
