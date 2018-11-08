/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb/errors"
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
		Witness:           make([]byte, 0),
		ExtraData:         make([]byte, 0),
	}
}

func Test_blockchainDatabase_Header_invalid(t *testing.T) {
	header := newTestBlockHeader()
	headerHash := header.Hash()

	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()

	bcStore.PutBlockHeader(headerHash, header, header.Difficulty, true)

	// Invalid block height
	hash1, err1 := bcStore.GetBlockHash(10)
	assert.Equal(t, err1 != nil, true)
	assert.Equal(t, hash1, common.EmptyHash)

	// Invalid block header hash
	_, err2 := bcStore.GetBlockHeader(common.StringToHash("heh"))
	assert.Equal(t, err2 != nil, true)

	// Invalid block header hash
	_, err3 := bcStore.GetBlockTotalDifficulty(common.StringToHash("heh"))
	assert.Equal(t, err3 != nil, true)

	// Invalid block hash
	block2 := newTestFullBlock(3, 3)
	_, err4 := bcStore.GetBlock(block2.HeaderHash)
	assert.Equal(t, err4 != nil, true)

	// Invalid block
	var block1 *types.Block
	assert.Panics(t, func() { bcStore.PutBlock(block1, header.Difficulty, true) }, "block is nil")
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

	td, err := bcStore.GetBlockTotalDifficulty(headerHash)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, td, header.Difficulty)

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
			From:     *crypto.MustGenerateRandomAddress(),
			To:       *crypto.MustGenerateRandomAddress(),
			Amount:   big.NewInt(3),
			GasPrice: big.NewInt(0),
			Payload:  make([]byte, 0),
		},
		Signature: crypto.Signature{Sig: []byte("test sig")},
	}

	tx.Hash = crypto.MustHash(tx.Data)

	return tx
}

func newTestDebt() *types.Debt {
	return types.NewDebtWithContext(newTestTx())
}

func Test_blockchainDatabase_Block(t *testing.T) {
	block := newTestFullBlock(3, 3)

	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()

	err := bcStore.PutBlock(block, block.Header.Difficulty, true)
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
}

func Test_blockchainDatabase_Receipt(t *testing.T) {
	block := newTestFullBlock(3, 3)

	receipts := []*types.Receipt{
		&types.Receipt{TxHash: block.Transactions[0].Hash},
		&types.Receipt{TxHash: block.Transactions[1].Hash},
		&types.Receipt{TxHash: block.Transactions[2].Hash},
	}

	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()

	if err := bcStore.PutBlock(block, block.Header.Difficulty, true); err != nil {
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

func Test_blockchainDatabase_GetTxIndex(t *testing.T) {
	block := newTestFullBlock(3, 3)

	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()

	err := bcStore.PutBlock(block, block.Header.Difficulty, true)
	assert.Equal(t, err, error(nil))

	for i, tx := range block.Transactions {
		txIdx, err := bcStore.GetTxIndex(tx.Hash)
		assert.Equal(t, err, error(nil))
		assert.Equal(t, txIdx != nil, true)
		assert.Equal(t, txIdx.Index, uint(i))
		assert.Equal(t, txIdx.BlockHash, block.HeaderHash)
	}

	// tx that doesn't exist
	txNoExist := newTestTx()
	_, err = bcStore.GetTxIndex(txNoExist.Hash)
	assert.Equal(t, err != nil, true)
}

func Test_blockchainDatabase_GetDebtIndex(t *testing.T) {
	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()
	GetDebtIndexTest(t, bcStore)
}

func GetDebtIndexTest(t *testing.T, bcStore BlockchainStore) {
	block := newTestFullBlock(3, 3)

	err := bcStore.PutBlock(block, block.Header.Difficulty, true)
	assert.Equal(t, err, error(nil))

	for i, d := range block.Debts {
		debtIndex, err := bcStore.GetDebtIndex(d.Hash)
		assert.Equal(t, err, error(nil))
		assert.Equal(t, debtIndex != nil, true)
		assert.Equal(t, debtIndex.Index, uint(i))
		assert.Equal(t, debtIndex.BlockHash, block.HeaderHash)
	}

	// tx that doesn't exist
	debtNoExist := newTestDebt()
	_, err = bcStore.GetTxIndex(debtNoExist.Hash)
	assert.Equal(t, err != nil, true)
}

func Test_blockchainDatabase_AddIndices(t *testing.T) {
	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()

	block := newTestFullBlock(3, 3)
	assert.Nil(t, bcStore.AddIndices(block))

	for i, tx := range block.Transactions {
		idx, _ := bcStore.GetTxIndex(tx.Hash)
		assert.Equal(t, idx.BlockHash, block.HeaderHash)
		assert.Equal(t, idx.Index, uint(i))
	}

	for i, d := range block.Debts {
		idx, _ := bcStore.GetDebtIndex(d.Hash)
		assert.Equal(t, idx.BlockHash, block.HeaderHash)
		assert.Equal(t, idx.Index, uint(i))
	}
}

func Test_blockchainDatabase_DeleteIndices(t *testing.T) {
	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()

	block := newTestFullBlock(3, 3)
	assert.Nil(t, bcStore.AddIndices(block))
	assert.Nil(t, bcStore.DeleteIndices(block))

	for _, tx := range block.Transactions {
		_, err := bcStore.GetTxIndex(tx.Hash)
		assert.Equal(t, errors.ErrNotFound, err)
	}

	for _, d := range block.Debts {
		_, err := bcStore.GetDebtIndex(d.Hash)
		assert.Equal(t, errors.ErrNotFound, err)
	}
}

func Test_blockchainDatabase_DeleteIndices_BlockHashMismatch(t *testing.T) {
	bcStore, dispose := newTestBlockchainDatabase()
	defer dispose()

	txs := []*types.Transaction{newTestTx(), newTestTx(), newTestTx()}
	debts := []*types.Debt{newTestDebt(), newTestDebt(), newTestDebt()}

	// old canonical chain
	block1 := &types.Block{
		HeaderHash:   common.StringToHash("block 1"),
		Transactions: txs[0:2],   // tx0, tx1
		Debts:        debts[0:2], // debt0, debt1
	}
	assert.Nil(t, bcStore.AddIndices(block1))

	// new canonical chain
	block2 := &types.Block{
		HeaderHash:   common.StringToHash("block 2"),
		Transactions: txs[1:3],   // tx1, tx2
		Debts:        debts[1:3], // debt1, debt2
	}

	// reorg the canonical chain
	assert.Nil(t, bcStore.AddIndices(block2))    // will update tx1's index
	assert.Nil(t, bcStore.DeleteIndices(block1)) // will not delete the tx1's index

	// tx0/debt0 's index deleted
	_, err := bcStore.GetTxIndex(txs[0].Hash)
	assert.Equal(t, errors.ErrNotFound, err)
	_, err = bcStore.GetDebtIndex(debts[0].Hash)
	assert.Equal(t, errors.ErrNotFound, err)

	// tx1/tx2/debt1/debt/2 exists
	txIdx1, _ := bcStore.GetTxIndex(txs[1].Hash)
	assert.Equal(t, txIdx1.BlockHash, common.StringToHash("block 2"))
	txIdx2, _ := bcStore.GetTxIndex(txs[2].Hash)
	assert.Equal(t, txIdx2.BlockHash, common.StringToHash("block 2"))
	debtIdx1, _ := bcStore.GetDebtIndex(debts[1].Hash)
	assert.Equal(t, debtIdx1.BlockHash, common.StringToHash("block 2"))
	debtIdx2, _ := bcStore.GetDebtIndex(debts[2].Hash)
	assert.Equal(t, debtIdx2.BlockHash, common.StringToHash("block 2"))
}
