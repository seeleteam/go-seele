/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

var (
	testBlockHash = common.StringToHash("block hash")
)

func Test_cachedStore_GetBlockHash(t *testing.T) {
	store := NewMemStore()
	store.PutBlockHash(38, testBlockHash)

	cachedStore := NewCachedStore(store)

	// key not found
	hash, _ := cachedStore.GetBlockHash(2)
	assert.Equal(t, hash, common.EmptyHash)

	// key found
	hash, err := cachedStore.GetBlockHash(38)
	assert.Equal(t, err, nil)
	assert.Equal(t, hash, testBlockHash)

	// key cached
	store.DeleteBlockHash(38)
	hash, _ = cachedStore.GetBlockHash(38)
	assert.Equal(t, hash, testBlockHash)
}

func Test_cachedStore_PutBlockHash(t *testing.T) {
	store := NewMemStore()
	cachedStore := NewCachedStore(store)

	err := cachedStore.PutBlockHash(38, testBlockHash)
	assert.Equal(t, err, nil)

	hash, _ := store.GetBlockHash(38)
	assert.Equal(t, hash, testBlockHash)

	// key cached
	store.DeleteBlockHash(38)
	hash, _ = cachedStore.GetBlockHash(38)
	assert.Equal(t, hash, testBlockHash)
}

func Test_cachedStore_DeleteBlockHash(t *testing.T) {
	store := NewMemStore()
	cachedStore := NewCachedStore(store)

	err := cachedStore.PutBlockHash(38, testBlockHash)
	assert.Equal(t, err, nil)

	cachedStore.DeleteBlockHash(38)
	hash, _ := cachedStore.GetBlockHash(38)
	assert.Equal(t, hash, common.EmptyHash)
}

func Test_cachedStore_GetBlockHeader(t *testing.T) {
	store := NewMemStore()
	header := newTestBlockHeader()
	hash := header.Hash()
	store.PutBlockHeader(hash, header, big.NewInt(38), false)

	cachedStore := NewCachedStore(store)

	// key not found
	header2, _ := cachedStore.GetBlockHeader(common.StringToHash("block hash 2"))
	assert.Equal(t, header2, (*types.BlockHeader)(nil))

	// key found
	header2, _ = cachedStore.GetBlockHeader(hash)
	assert.Equal(t, header2, header)

	// key cached
	store.DeleteBlock(hash)
	header2, _ = cachedStore.GetBlockHeader(hash)
	assert.Equal(t, header2, header)
}

func Test_cachedStore_PutBlockHeader(t *testing.T) {
	store := NewMemStore()
	cachedStore := NewCachedStore(store)

	header := newTestBlockHeader()
	hash := header.Hash()
	err := cachedStore.PutBlockHeader(hash, header, big.NewInt(38), true)
	assert.Equal(t, err, nil)

	header2, _ := cachedStore.GetBlockHeader(hash)
	assert.Equal(t, header2, header)

	// key cached
	store.DeleteBlock(hash)
	header2, _ = cachedStore.GetBlockHeader(hash)
	assert.Equal(t, header2, header)
	td, _ := cachedStore.GetBlockTotalDifficulty(hash)
	assert.Equal(t, td, big.NewInt(38))
	hash2, _ := cachedStore.GetBlockHash(header.Height)
	assert.Equal(t, hash2, hash)
}

func Test_cachedStore_GetBlockTotalDifficulty(t *testing.T) {
	store := NewMemStore()
	header := newTestBlockHeader()
	hash := header.Hash()
	store.PutBlockHeader(hash, header, big.NewInt(38), false)
	cachedStore := NewCachedStore(store)

	td, _ := cachedStore.GetBlockTotalDifficulty(hash)
	assert.Equal(t, td, big.NewInt(38))

	// key cached
	store.DeleteBlock(hash)
	td, _ = cachedStore.GetBlockTotalDifficulty(hash)
	assert.Equal(t, td, big.NewInt(38))
}

func Test_cachedStore_PutBlock(t *testing.T) {
	store := NewMemStore()
	cachedStore := NewCachedStore(store)

	block := types.NewBlock(newTestBlockHeader(), []*types.Transaction{newTestTx()}, []*types.Receipt{&types.Receipt{}}, nil)
	err := cachedStore.PutBlock(block, big.NewInt(38), true)
	assert.Equal(t, err, nil)

	block2, _ := cachedStore.GetBlock(block.HeaderHash)
	assert.Equal(t, block2, block)

	// key cached
	store.DeleteBlock(block.HeaderHash)
	block2, _ = cachedStore.GetBlock(block.HeaderHash)
	assert.Equal(t, block2, block)
	header, _ := cachedStore.GetBlockHeader(block.HeaderHash)
	assert.Equal(t, header, block.Header)
	td, _ := cachedStore.GetBlockTotalDifficulty(block.HeaderHash)
	assert.Equal(t, td, big.NewInt(38))
	hash, _ := cachedStore.GetBlockHash(block.Header.Height)
	assert.Equal(t, hash, block.HeaderHash)
}

func Test_cachedStore_GutBlock(t *testing.T) {
	store := NewMemStore()
	block := types.NewBlock(newTestBlockHeader(), []*types.Transaction{newTestTx()}, []*types.Receipt{&types.Receipt{}}, nil)
	store.PutBlock(block, big.NewInt(38), true)
	cachedStore := NewCachedStore(store)

	block2, _ := cachedStore.GetBlock(block.HeaderHash)
	assert.Equal(t, block2, block)

	// key cached
	store.DeleteBlock(block.HeaderHash)
	block2, _ = cachedStore.GetBlock(block.HeaderHash)
	assert.Equal(t, block2, block)
}

func Test_cachedStore_DeleteBlock(t *testing.T) {
	store := NewMemStore()
	cachedStore := NewCachedStore(store)

	block := types.NewBlock(newTestBlockHeader(), []*types.Transaction{newTestTx()}, []*types.Receipt{&types.Receipt{}}, nil)
	cachedStore.PutBlock(block, big.NewInt(38), true)

	assert.Equal(t, cachedStore.DeleteBlock(block.HeaderHash), nil)

	// key not cached anymore
	header, _ := cachedStore.GetBlockHeader(block.HeaderHash)
	assert.Equal(t, header, (*types.BlockHeader)(nil))
	td, _ := cachedStore.GetBlockTotalDifficulty(block.HeaderHash)
	assert.Equal(t, td, (*big.Int)(nil))
	block2, _ := cachedStore.GetBlock(block.HeaderHash)
	assert.Equal(t, block2, (*types.Block)(nil))
}

func Test_GetDebtIndex(t *testing.T) {
	store := NewMemStore()
	cachedStore := NewCachedStore(store)

	GetDebtIndexTest(t, cachedStore)
}
