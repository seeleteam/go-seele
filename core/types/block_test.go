/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package types

import (
	"math/big"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/stretchr/testify/assert"
)

func newTestBlockHeader(t *testing.T) *BlockHeader {
	return &BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           randomAddress(t),
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            1,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Consensus:         PowConsensus,
		Witness:           common.CopyBytes([]byte("witness")),
		ExtraData:         common.CopyBytes([]byte("ExtraData")),
	}
}

func Test_BlockMarshal(t *testing.T) {
	header := newTestBlockHeader(t)

	txs := []*Transaction{
		newTestTx(t, 10, 1, 1, true),
		newTestTx(t, 20, 1, 2, true),
		newTestTx(t, 30, 1, 3, true),
	}
	receipts := []*Receipt{
		newTestReceipt(),
		newTestReceipt(),
		newTestReceipt(),
	}

	tx1 := newTestTx(t, 1, 1, 1, true)
	d1 := NewDebtWithContext(tx1)
	debts := []*Debt{
		d1,
	}

	block := NewBlock(header, txs, receipts, debts)

	common.SerializePanic(block)
}

func Test_BlockHeaderMarshal(t *testing.T) {
	header := newTestBlockHeader(t)
	common.SerializePanic(header)
}

func Test_BlockHeader_Clone(t *testing.T) {
	header := newTestBlockHeader(t)
	cloned := header.Clone()

	originalAddress := header.Creator
	originalTimestamp := header.CreateTimestamp.Int64()

	// Change all values of original header.
	header.PreviousBlockHash = common.StringToHash("PreviousBlockHash2")
	header.Creator = randomAddress(t)
	header.StateHash = crypto.HashBytes([]byte("StateHash2"))
	header.TxHash = crypto.HashBytes([]byte("TxHash2"))
	header.Difficulty.SetInt64(2)
	header.Height = 2
	header.CreateTimestamp.SetInt64(2)
	header.Witness = common.CopyBytes([]byte("witness2"))
	header.ExtraData = common.CopyBytes([]byte("ExtraData2"))

	// Ensure the cloned header is not affected.
	assert.Equal(t, cloned.PreviousBlockHash, common.StringToHash("PreviousBlockHash"))
	assert.Equal(t, cloned.Creator, originalAddress)
	assert.Equal(t, cloned.StateHash, common.StringToHash("StateHash"))
	assert.Equal(t, cloned.TxHash, common.StringToHash("TxHash"))
	assert.Equal(t, cloned.Difficulty.Int64(), int64(1))
	assert.Equal(t, cloned.Height, uint64(1))
	assert.Equal(t, cloned.CreateTimestamp.Int64(), originalTimestamp)
	assert.Equal(t, cloned.ExtraData, []byte("ExtraData"))
	assert.Equal(t, cloned.Witness, []byte("witness"))
}

func Test_BlockHeader_Hash(t *testing.T) {
	header := newTestBlockHeader(t)
	hash1 := header.Hash()

	header.Height = 2
	hash2 := header.Hash()

	assert.Equal(t, hash1.Equal(hash2), false)
}

func Test_Block_NewBlock(t *testing.T) {
	header := newTestBlockHeader(t)
	txs := []*Transaction{
		newTestTx(t, 10, 1, 1, true),
		newTestTx(t, 20, 1, 2, true),
		newTestTx(t, 30, 1, 3, true),
	}
	receipts := []*Receipt{
		newTestReceipt(),
		newTestReceipt(),
		newTestReceipt(),
	}

	block := NewBlock(header, txs, receipts, nil)
	assert.Equal(t, block != nil, true)

	// ensure the header is copied
	header.TxHash = common.StringToHash("ChangedTxHash")
	header.ReceiptHash = common.StringToHash("ChangedReceiptHash")
	assert.Equal(t, block.Header.TxHash != header.TxHash, true)
	assert.Equal(t, block.Header.ReceiptHash != header.ReceiptHash, true)
	assert.Equal(t, block.Header.TxHash == MerkleRootHash(txs), true)
	assert.Equal(t, block.Header.ReceiptHash == ReceiptMerkleRootHash(receipts), true)

	// verify HeaderHash
	assert.Equal(t, block.HeaderHash == block.Header.Hash(), true)
}

func Test_Block_GetExcludeRewardTransactions(t *testing.T) {
	header := newTestBlockHeader(t)
	txs := []*Transaction{
		newTestTx(t, 10, 1, 1, true),
		newTestTx(t, 20, 1, 2, true),
		newTestTx(t, 30, 1, 3, true),
	}

	block := NewBlock(header, txs, nil, nil)
	excludeTxs := block.GetExcludeRewardTransactions()
	assert.Equal(t, len(excludeTxs) == 2, true)

	// only reward transaction
	rewardTxs := []*Transaction{newTestTx(t, 10, 1, 1, true)}
	block = NewBlock(header, rewardTxs, nil, nil)
	excludeTxs = block.GetExcludeRewardTransactions()
	assert.Equal(t, len(excludeTxs) == 0, true)

	// txs is nil
	block = NewBlock(header, nil, nil, nil)
	excludeTxs = block.GetExcludeRewardTransactions()
	assert.Equal(t, len(excludeTxs) == 0, true)
}

func Test_Block_FindTransaction(t *testing.T) {
	header := newTestBlockHeader(t)
	txs := []*Transaction{
		newTestTx(t, 10, 1, 1, true),
		newTestTx(t, 20, 1, 2, true),
		newTestTx(t, 30, 1, 3, true),
	}

	block := NewBlock(header, txs, nil, nil)

	assert.Equal(t, block.FindTransaction(txs[0].Hash), txs[0])
	assert.Equal(t, block.FindTransaction(txs[1].Hash), txs[1])
	assert.Equal(t, block.FindTransaction(txs[2].Hash), txs[2])

	invalidHash := common.StringToHash("5aaeb6053f3e94c9b9a09f33669485e0")
	assert.Equal(t, block.FindTransaction(invalidHash), (*Transaction)(nil))
}

func Test_Block_GetShardNumber(t *testing.T) {
	// header is nil
	header := newTestBlockHeader(t)
	block := NewBlock(header, nil, nil, nil)
	block.Header = nil
	assert.Equal(t, block.GetShardNumber(), common.UndefinedShardNumber)

	// valid header with Creator in shard #1
	block.Header = newTestBlockHeader(t)
	addr := common.BigToAddress(big.NewInt(1))
	block.Header.Creator = addr
	assert.Equal(t, block.GetShardNumber(), uint(1))
}
