/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

func testBlockchain(t *testing.T, ut func(*Blockchain)) {
	testBlockchainDatabase(func(bcStore store.BlockchainStore) {
		genesis := DefaultGenesis(bcStore)
		if err := genesis.Initialize(); err != nil {
			t.Fatal()
		}

		bc, err := NewBlockchain(bcStore)
		if err != nil {
			t.Fatal()
		}

		ut(bc)
	})
}

func newTestBlock(t *testing.T, parent *types.Block) *types.Block {
	txs := []*types.Transaction{
		newTestTx(t, 1, 1),
		newTestTx(t, 2, 2),
		newTestTx(t, 3, 3),
	}

	header := &types.BlockHeader{
		PreviousBlockHash: parent.HeaderHash,
		Creator:           *crypto.MustGenerateRandomAddress(),
		TxHash:            types.MerkleRootHash(txs),
		Height:            parent.Header.Height + 1,
		Difficulty:        big.NewInt(3),
		CreateTimestamp:   big.NewInt(1),
		Nonce:             10,
	}

	return &types.Block{
		HeaderHash:   header.Hash(),
		Header:       header,
		Transactions: txs,
	}
}

func Test_Blockchain_WriteBlock_HeaderHashChanged(t *testing.T) {
	testBlockchain(t, func(bc *Blockchain) {
		newBlock := newTestBlock(t, bc.genesisBlock)
		newBlock.HeaderHash = common.EmptyHash

		err := bc.WriteBlock(newBlock)
		assert.Equal(t, err, ErrBlockHashMismatch)
	})
}

func Test_Blockchain_WriteBlock_TxRootHashChanged(t *testing.T) {
	testBlockchain(t, func(bc *Blockchain) {
		newBlock := newTestBlock(t, bc.genesisBlock)
		newBlock.Header.TxHash = common.EmptyHash
		newBlock.HeaderHash = newBlock.Header.Hash()

		err := bc.WriteBlock(newBlock)
		assert.Equal(t, err, ErrBlockTxsHashMismatch)
	})
}

func Test_Blockchain_WriteBlock_InvalidHeader(t *testing.T) {
	testBlockchain(t, func(bc *Blockchain) {
		newBlock := newTestBlock(t, bc.genesisBlock)
		newBlock.Header.Height = 10
		newBlock.HeaderHash = newBlock.Header.Hash()

		err := bc.WriteBlock(newBlock)
		assert.Equal(t, err, ErrHeaderChainInvalidHeight)
	})
}

func Test_Blockchain_WriteBlock_ValidBlock(t *testing.T) {
	testBlockchain(t, func(bc *Blockchain) {
		newBlock := newTestBlock(t, bc.genesisBlock)
		err := bc.WriteBlock(newBlock)
		assert.Equal(t, err, error(nil))
		assert.Equal(t, bc.currentBlock, newBlock)

		storedBlock, err := bc.bcStore.GetBlock(newBlock.HeaderHash)
		assert.Equal(t, err, error(nil))
		assert.Equal(t, storedBlock, newBlock)
	})
}

func Test_Blockchain_WriteBlock_DupBlocks(t *testing.T) {
	testBlockchain(t, func(bc *Blockchain) {
		newBlock := newTestBlock(t, bc.genesisBlock)

		err := bc.WriteBlock(newBlock)
		assert.Equal(t, err, error(nil))
		assert.Equal(t, bc.currentBlock, newBlock)

		err = bc.WriteBlock(newBlock)
		assert.Equal(t, err, ErrHeaderChainInvalidParentHash)
	})
}

func Test_Blockchain_WriteBlock_InsertTwoBlocks(t *testing.T) {
	testBlockchain(t, func(bc *Blockchain) {
		block1 := newTestBlock(t, bc.genesisBlock)
		err := bc.WriteBlock(block1)
		assert.Equal(t, err, error(nil))
		assert.Equal(t, bc.currentBlock, block1)

		block2 := newTestBlock(t, block1)
		err = bc.WriteBlock(block2)
		assert.Equal(t, err, error(nil))
		assert.Equal(t, bc.currentBlock, block2)
	})
}
