/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/stretchr/testify/assert"
)

func Test_BlockLeaf_Add_Remove(t *testing.T) {
	bf := NewBlockLeaves()

	index := NewBlockIndex(common.StringToHash("block"), big.NewInt(1))
	bf.Add(index)
	assert.Equal(t, bf.blockIndexMap.Count(), 1)

	index2 := NewBlockIndex(common.StringToHash("block 2"), big.NewInt(2))
	bf.Add(index2)
	assert.Equal(t, bf.blockIndexMap.Count(), 2)

	bf.RemoveByHash(index.blockHash)
	assert.Equal(t, bf.blockIndexMap.Count(), 1)

	bf.Remove(index)
	assert.Equal(t, bf.blockIndexMap.Count(), 1)

	bf.Remove(index2)
	assert.Equal(t, bf.blockIndexMap.Count(), 0)
}

func Test_BlockLeaf_Get(t *testing.T) {
	bf := NewBlockLeaves()
	index := NewBlockIndex(common.StringToHash("block"), big.NewInt(1))
	bf.Add(index)
	index2 := NewBlockIndex(common.StringToHash("block 2"), big.NewInt(2))
	bf.Add(index2)

	assert.Equal(t, bf.GetBestBlockIndex(), index2)

	assert.Equal(t, bf.GetBlockIndexByHash(index.blockHash), index)

	index3 := NewBlockIndex(common.StringToHash("block 3"), big.NewInt(2))
	assert.Equal(t, bf.IsBestBlockIndex(index3), false)

	index4 := NewBlockIndex(common.StringToHash("block 4"), big.NewInt(3))
	assert.Equal(t, bf.IsBestBlockIndex(index4), true)
}
