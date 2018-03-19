/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package types

import (
	"math/big"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

func newTestBlockHeader(t *testing.T) *BlockHeader {
	return &BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           randomAddress(t),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            1,
		CreateTimestamp:   big.NewInt(time.Now().UnixNano()),
		Nonce:             1,
	}
}

func Test_BlockHeader_Clone(t *testing.T) {
	header := newTestBlockHeader(t)
	cloned := header.Clone()

	originalAddress := header.Creator
	originalTimestamp := header.CreateTimestamp.Int64()

	// Change all values of original header.
	header.PreviousBlockHash = common.StringToHash("PreviousBlockHash2")
	header.Creator = randomAddress(t)
	header.TxHash = common.BytesToHash(crypto.Keccak256Hash([]byte("TxHash2")))
	header.Difficulty.SetInt64(2)
	header.Height = 2
	header.CreateTimestamp.SetInt64(2)
	header.Nonce = 2

	// Ensure the cloned header is not affected.
	assert.Equal(t, cloned.PreviousBlockHash, common.StringToHash("PreviousBlockHash"))
	assert.Equal(t, cloned.Creator, originalAddress)
	assert.Equal(t, cloned.TxHash, common.StringToHash("TxHash"))
	assert.Equal(t, cloned.Difficulty.Int64(), int64(1))
	assert.Equal(t, cloned.Height, uint64(1))
	assert.Equal(t, cloned.CreateTimestamp.Int64(), originalTimestamp)
	assert.Equal(t, cloned.Nonce, uint64(1))
}

func Test_BlockHeader_Hash(t *testing.T) {
	header := newTestBlockHeader(t)
	hash1 := header.Hash()

	header.Nonce = 2
	hash2 := header.Hash()

	assert.Equal(t, hash1.Equal(hash2), false)
}

func Test_Block_FindTransaction(t *testing.T) {
	header := newTestBlockHeader(t)
	txs := []*Transaction{
		newTestTx(t, 10, 1, true),
		newTestTx(t, 20, 2, true),
		newTestTx(t, 30, 3, true),
	}

	block := NewBlock(header, txs)

	assert.Equal(t, block.FindTransaction(txs[0].Hash), txs[0])
	assert.Equal(t, block.FindTransaction(txs[1].Hash), txs[1])

	invalidHash := common.StringToHash("5aaeb6053f3e94c9b9a09f33669485e0")
	assert.Equal(t, block.FindTransaction(invalidHash), (*Transaction)(nil))
}
