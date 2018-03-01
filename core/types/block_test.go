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
		PreviousBlockHash: common.StringToHash("5aaeb6053f3e94c9b9a09f33669435e0"),
		Creator:           randomAddress(t),
		TxHash:            common.BytesToHash(crypto.Keccak256Hash([]byte("test"))),
		Difficulty:        big.NewInt(1),
		Height:            big.NewInt(1),
		CreateTimestamp:   big.NewInt(time.Now().UnixNano()),
		Nonce:             1,
	}
}

func Test_BlockHeader_Clone(t *testing.T) {
	header := newTestBlockHeader(t)
	cloned := header.Clone()

	// Change original header, including value type and pointer type.
	header.Nonce = 2
	header.Height.SetInt64(2)

	// Ensure the cloned header is not affected.
	assert.Equal(t, cloned.Nonce, uint64(1))
	assert.Equal(t, cloned.Height.Int64(), int64(1))
}

func Test_Block_FindTransaction(t *testing.T) {
	header := newTestBlockHeader(t)
	txs := []*Transaction{
		newTestTx(t, 10, 1),
		newTestTx(t, 20, 2),
		newTestTx(t, 30, 3),
	}

	block := NewBlock(header, txs)

	assert.Equal(t, block.FindTransaction(txs[0].Hash), txs[0])
	assert.Equal(t, block.FindTransaction(txs[1].Hash), txs[1])

	invalidHash := common.StringToHash("5aaeb6053f3e94c9b9a09f33669485e0")
	assert.Equal(t, block.FindTransaction(invalidHash), (*Transaction)(nil))
}
