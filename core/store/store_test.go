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

func testBlockchainDatabase(ut func(BlockchainStore)) {
	dir, err := ioutil.TempDir("", "BlockchainStore")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	db, err := leveldb.NewLevelDB(dir)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	ut(NewBlockchainDatabase(db))
}

func newTestBlockHeader(t *testing.T) *types.BlockHeader {
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
	header := newTestBlockHeader(t)
	headerHash := header.Hash()

	testBlockchainDatabase(func(bcStore BlockchainStore) {
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
	})
}

func newTestTx() *types.Transaction {
	return &types.Transaction{
		Hash: common.EmptyHash,
		Data: &types.TransactionData{
			From:   *crypto.MustGenerateRandomAddress(),
			To:     crypto.MustGenerateRandomAddress(),
			Amount: big.NewInt(3),
		},
		Signature: &crypto.Signature{big.NewInt(1), big.NewInt(2)},
	}
}

func Test_blockchainDatabase_Block(t *testing.T) {
	header := newTestBlockHeader(t)
	block := &types.Block{
		HeaderHash:   header.Hash(),
		Header:       header,
		Transactions: []*types.Transaction{newTestTx(), newTestTx(), newTestTx()},
	}

	testBlockchainDatabase(func(bcStore BlockchainStore) {
		err := bcStore.PutBlock(block, header.Difficulty, true)
		assert.Equal(t, err, error(nil))

		storedBlock, err := bcStore.GetBlock(block.HeaderHash)
		assert.Equal(t, err, error(nil))
		assert.Equal(t, storedBlock, block)
	})
}
