/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func newTestBlockchainDatabase() (BlockchainStore, func()) {
	dir, err := ioutil.TempDir("", "BlockchainStore")
	if err != nil {
		panic(err)
	}

	db, err := leveldb.NewLevelDB(dir)
	if err != nil {
		defer os.RemoveAll(dir)
		panic(err)
	}

	return NewBlockchainDatabase(db), func() {
		defer db.Close()
		defer os.RemoveAll(dir)
	}
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

	headHash, err := bcStore.GetHeadBlockHash()
	assert.Equal(t, err, error(nil))
	assert.Equal(t, headHash, headerHash)

	storedHeader, err := bcStore.GetBlockHeader(headerHash)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, storedHeader.Hash(), headerHash)

	td, err := bcStore.GetBlockTotalDifficulty(headerHash)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, td, header.Difficulty)

	exist, err := bcStore.HasBlock(headerHash)
	assert.Equal(t, exist, true)
	assert.Equal(t, err, nil)

	exist, err = bcStore.HasBlock(common.EmptyHash)
	assert.Equal(t, exist, false)
	assert.Equal(t, err, nil)
}

func newTestTx() *types.Transaction {
	tx := &types.Transaction{
		Data: &types.TransactionData{
			From:    *crypto.MustGenerateRandomAddress(),
			To:      crypto.MustGenerateRandomAddress(),
			Amount:  big.NewInt(3),
			Payload: make([]byte, 0),
		},
		Signature: &crypto.Signature{big.NewInt(1), big.NewInt(2)},
	}

	tx.Hash = crypto.MustHash(tx)

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
