/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/core/state"

	"github.com/seeleteam/go-seele/database"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

func newTestBlockchain(db database.Database) *Blockchain {
	bcStore := store.NewBlockchainDatabase(db)

	genesis := DefaultGenesis(bcStore)
	if err := genesis.Initialize(db); err != nil {
		panic(err)
	}

	bc, err := NewBlockchain(bcStore, db)
	if err != nil {
		panic(err)
	}

	return bc
}

func newTestBlock(t *testing.T, parent *types.Block, db database.Database) *types.Block {
	txs := []*types.Transaction{
		newTestTx(t, 1, 1),
		newTestTx(t, 2, 2),
		newTestTx(t, 3, 3),
	}

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		t.Fatal()
	}

	for _, tx := range txs {
		stateObj := statedb.GetOrNewStateObject(tx.Data.From)
		stateObj.SetAmount(big.NewInt(10))
		stateObj.SetNonce(1)
	}

	batch := db.NewBatch()
	stateHash, err := statedb.Commit(batch)
	if err != nil {
		t.Fatal()
	}

	if err = batch.Commit(); err != nil {
		t.Fatal()
	}

	header := &types.BlockHeader{
		PreviousBlockHash: parent.HeaderHash,
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         stateHash,
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
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(t, bc.genesisBlock, db)
	newBlock.HeaderHash = common.EmptyHash

	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, ErrBlockHashMismatch)
}

func Test_Blockchain_WriteBlock_TxRootHashChanged(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(t, bc.genesisBlock, db)
	newBlock.Header.TxHash = common.EmptyHash
	newBlock.HeaderHash = newBlock.Header.Hash()

	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, ErrBlockTxsHashMismatch)
}

func Test_Blockchain_WriteBlock_InvalidHeader(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(t, bc.genesisBlock, db)
	newBlock.Header.Height = 10
	newBlock.HeaderHash = newBlock.Header.Hash()

	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, ErrHeaderChainInvalidHeight)
}

func Test_Blockchain_WriteBlock_ValidBlock(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(t, bc.genesisBlock, db)
	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, bc.currentBlock, newBlock)

	storedBlock, err := bc.bcStore.GetBlock(newBlock.HeaderHash)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, storedBlock, newBlock)

	_, err = state.NewStatedb(newBlock.Header.StateHash, db)
	assert.Equal(t, err, error(nil))
}

func Test_Blockchain_WriteBlock_DupBlocks(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(t, bc.genesisBlock, db)

	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, bc.currentBlock, newBlock)

	err = bc.WriteBlock(newBlock)
	assert.Equal(t, err, ErrHeaderChainInvalidParentHash)
}

func Test_Blockchain_WriteBlock_InsertTwoBlocks(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block1 := newTestBlock(t, bc.genesisBlock, db)
	err := bc.WriteBlock(block1)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, bc.currentBlock, block1)

	block2 := newTestBlock(t, block1, db)
	err = bc.WriteBlock(block2)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, bc.currentBlock, block2)
}
