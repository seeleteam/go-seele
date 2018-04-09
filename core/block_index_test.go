/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/core/types"
)

func getTestBlock(t *testing.T, difficult int64) *types.Block {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)
	return newTestBlock(bc.genesisBlock.HeaderHash, 1, 3, 0)
}

func Test_BlockLeaf_Add_Remove(t *testing.T) {
	bf := NewBlockLeaf()

	index := NewBlockIndex(nil, getTestBlock(t, 1), big.NewInt(1))
	bf.Add(index)
	assert.Equal(t, bf.blockIndexMap.Count(), 1)

	index2 := NewBlockIndex(nil, getTestBlock(t, 2), big.NewInt(2))
	bf.Add(index2)
	assert.Equal(t, bf.blockIndexMap.Count(), 2)

	bf.RemoveByHash(index.currentBlock.HeaderHash)
	assert.Equal(t, bf.blockIndexMap.Count(), 1)

	bf.Remove(index)
	assert.Equal(t, bf.blockIndexMap.Count(), 1)

	bf.Remove(index2)
	assert.Equal(t, bf.blockIndexMap.Count(), 0)
}

func Test_BlockLeaf_Get(t *testing.T) {
	bf := NewBlockLeaf()
	index := NewBlockIndex(nil, getTestBlock(t, 1), big.NewInt(1))
	bf.Add(index)
	index2 := NewBlockIndex(nil, getTestBlock(t, 2), big.NewInt(2))
	bf.Add(index2)

	assert.Equal(t, bf.GetBestBlockIndex(), index2)
	assert.Equal(t, bf.GetBestBlock(), index2.currentBlock)
	assert.Equal(t, bf.GetBestStateDB(), index2.state)

	assert.Equal(t, bf.GetBlockIndexByHash(index.currentBlock.HeaderHash), index)

	index3 := NewBlockIndex(nil, getTestBlock(t, 2), big.NewInt(2))
	assert.Equal(t, bf.IsBestBlockIndex(index3), false)

	index4 := NewBlockIndex(nil, getTestBlock(t, 3), big.NewInt(3))
	assert.Equal(t, bf.IsBestBlockIndex(index4), true)
}
