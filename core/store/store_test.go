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
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func newTestBlockchainDatabase() (BlockchainStore, func()) {
	db, dispose := leveldb.NewTestDatabase()
	return NewBlockchainDatabase(db), dispose
}

func newTestBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            1,
		CreateTimestamp:   big.NewInt(1),
		Nonce:             1,
		ExtraData:         make([]byte, 0),
	}
}

func Test_blockchainDatabase_Header(t *testing.T) {
	header := newTestBlockHeader()
	headerHash := header.Hash()

	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()

	bcStore.PutBlockHeader(headerHash, header, header.Difficulty, true)

	hash, err := bcStore.GetBlockHash(1)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, hash, headerHash)

	// Invaild block height
	hash1, err1 := bcStore.GetBlockHash(10)
	assert.Matches(t, err1.Error(), "not found")
	assert.Equal(t, hash1, common.EmptyHash)

	// PutBlockHash test
	err2 := bcStore.PutBlockHash(10, headerHash)
	assert.Equal(t, err2, error(nil))
	hash2, err2 := bcStore.GetBlockHash(10)
	assert.Equal(t, err2, error(nil))
	assert.Equal(t, hash2, headerHash)

	// DeleteBlockHash test
	exist, err3 := bcStore.DeleteBlockHash(10)
	assert.Equal(t, err3, error(nil))
	assert.Equal(t, exist, true)

	exist, err4 := bcStore.DeleteBlockHash(10)
	assert.Equal(t, err4, error(nil))
	assert.Equal(t, exist, false)

	headHash, err := bcStore.GetHeadBlockHash()
	assert.Equal(t, err, error(nil))
	assert.Equal(t, headHash, headerHash)

	// PutHeadBlockHash test
	err5 := bcStore.PutHeadBlockHash(headerHash)
	assert.Equal(t, err5, error(nil))

	storedHeader, err := bcStore.GetBlockHeader(headerHash)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, storedHeader.Hash(), headerHash)

	// Invalid block header hash
	_, err6 := bcStore.GetBlockHeader(common.StringToHash("heh"))
	assert.Matches(t, err6.Error(), "not found")

	td, err := bcStore.GetBlockTotalDifficulty(headerHash)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, td, header.Difficulty)

	// Invalid block header hash
	_, err7 := bcStore.GetBlockTotalDifficulty(common.StringToHash("heh"))
	assert.Matches(t, err7.Error(), "not found")

	exist, err = bcStore.HasBlock(headerHash)
	assert.Equal(t, exist, true)
	assert.Equal(t, err, nil)

	exist, err = bcStore.HasBlock(common.EmptyHash)
	assert.Equal(t, exist, false)
	assert.Equal(t, err, nil)
}

func newTestTx() *types.Transaction {
	tx := &types.Transaction{
		Data: types.TransactionData{
			From:    *crypto.MustGenerateRandomAddress(),
			To:      *crypto.MustGenerateRandomAddress(),
			Amount:  big.NewInt(3),
			Fee:     big.NewInt(0),
			Payload: make([]byte, 0),
		},
		Signature: crypto.Signature{Sig: []byte("test sig")},
	}

	tx.Hash = crypto.MustHash(tx.Data)

	return tx
}

func Test_blockchainDatabase_Block(t *testing.T) {
	header := newTestBlockHeader()
	block := &types.Block{
		HeaderHash:   header.Hash(),
		Header:       header,
		Transactions: []*types.Transaction{newTestTx(), newTestTx(), newTestTx()},
	}

	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()

	err := bcStore.PutBlock(block, header.Difficulty, true)
	assert.Equal(t, err, error(nil))

	storedBlock, err := bcStore.GetBlock(block.HeaderHash)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, storedBlock, block)

	// GetBlockByHeight test
	block3, err3 := bcStore.GetBlockByHeight(1)
	assert.Equal(t, err3, nil)
	assert.Equal(t, block3, block)

	// DeleteBlock test
	err4 := bcStore.DeleteBlock(block.HeaderHash)
	assert.Equal(t, err4, nil)

	// Invalid block hash
	block2 := block
	block2.HeaderHash = common.StringToHash("heh")
	_, err2 := bcStore.GetBlock(block2.HeaderHash)
	assert.Matches(t, err2.Error(), "not found")

	// Invalid block
	var block1 *types.Block
	defer func() {
		err1 := recover()
		err1str, ok := err1.(string)
		assert.Equal(t, ok, true)
		assert.Matches(t, err1str, "block is nil")
	}()
	bcStore.PutBlock(block1, header.Difficulty, true)
}

func Test_blockchainDatabase_Receipt(t *testing.T) {
	header := newTestBlockHeader()
	block := &types.Block{
		HeaderHash:   header.Hash(),
		Header:       header,
		Transactions: []*types.Transaction{newTestTx(), newTestTx(), newTestTx()},
	}

	receipts := []*types.Receipt{
		&types.Receipt{TxHash: block.Transactions[0].Hash},
		&types.Receipt{TxHash: block.Transactions[1].Hash},
		&types.Receipt{TxHash: block.Transactions[2].Hash},
	}

	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()

	if err := bcStore.PutBlock(block, header.Difficulty, true); err != nil {
		t.Fatal()
	}

	if err := bcStore.PutReceipts(block.HeaderHash, receipts); err != nil {
		t.Fatal()
	}

	// Check receipts in the block
	storedReceipts, err := bcStore.GetReceiptsByBlockHash(block.HeaderHash)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, len(storedReceipts), 3)

	// Check single receipt
	for i := 0; i < 3; i++ {
		txHash := block.Transactions[i].Hash
		receipt, err := bcStore.GetReceiptByTxHash(txHash)
		assert.Equal(t, err, error(nil))
		assert.Equal(t, receipt.TxHash, txHash)
	}
}
