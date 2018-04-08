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
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
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

func newTestBlock(t *testing.T, parentHash common.Hash, height uint64, db database.Database, nonce uint64, difficulty int64) *types.Block {
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
		stateObj.SetNonce(nonce)
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
		PreviousBlockHash: parentHash,
		Creator:           *crypto.MustGenerateRandomAddress(),
		StateHash:         stateHash,
		TxHash:            types.MerkleRootHash(txs),
		Height:            height,
		Difficulty:        big.NewInt(difficulty),
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

	newBlock := newTestBlock(t, bc.genesisBlock.HeaderHash, 1, db, 1, 3)
	newBlock.HeaderHash = common.EmptyHash

	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, ErrBlockHashMismatch)
}

func Test_Blockchain_WriteBlock_TxRootHashChanged(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(t, bc.genesisBlock.HeaderHash, 1, db, 1, 3)
	newBlock.Header.TxHash = common.EmptyHash
	newBlock.HeaderHash = newBlock.Header.Hash()

	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, ErrBlockTxsHashMismatch)
}

func Test_Blockchain_WriteBlock_InvalidHeader(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(t, bc.genesisBlock.HeaderHash, 1, db, 1, 3)
	newBlock.Header.Height = 10
	newBlock.HeaderHash = newBlock.Header.Hash()

	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, ErrBlockInvalidHeight)
}

func Test_Blockchain_WriteBlock_ValidBlock(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(t, bc.genesisBlock.HeaderHash, 1, db, 1, 3)
	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, error(nil))

	currentBlock, _ := bc.CurrentBlock()
	assert.Equal(t, currentBlock, newBlock)

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

	newBlock := newTestBlock(t, bc.genesisBlock.HeaderHash, 1, db, 1, 3)

	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, error(nil))

	currentBlock, _ := bc.CurrentBlock()
	assert.Equal(t, currentBlock, newBlock)

	err = bc.WriteBlock(newBlock)
	assert.Equal(t, err, ErrBlockAlreadyExist)
}

func Test_Blockchain_WriteBlock_InsertTwoBlocks(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block1 := newTestBlock(t, bc.genesisBlock.HeaderHash, 1, db, 1, 3)
	err := bc.WriteBlock(block1)
	assert.Equal(t, err, error(nil))

	currentBlock, _ := bc.CurrentBlock()
	assert.Equal(t, currentBlock, block1)

	block2 := newTestBlock(t, block1.HeaderHash, 2, db, 2, 3)
	err = bc.WriteBlock(block2)
	assert.Equal(t, err, error(nil))

	currentBlock, _ = bc.CurrentBlock()
	assert.Equal(t, currentBlock, block2)
}

func Test_Blockchain_BlockFork(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block1 := newTestBlock(t, bc.genesisBlock.HeaderHash, 1, db, 1, 3)
	err := bc.WriteBlock(block1)
	assert.Equal(t, err, error(nil))

	currentBlock, _ := bc.CurrentBlock()
	assert.Equal(t, currentBlock, block1)
	assert.Equal(t, bc.blockLeaves.Count(), 1)

	block2 := newTestBlock(t, bc.genesisBlock.HeaderHash, 1, db, 2, 3)
	err = bc.WriteBlock(block2)
	assert.Equal(t, err, error(nil))

	assert.Equal(t, bc.blockLeaves.Count(), 2)
}

func Test_BlockChain_InvalidParent(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block := newTestBlock(t, common.Hash{}, 1, db, 1, 3)

	err := bc.ValidateBlock(block)
	assert.Equal(t, err, ErrBlockInvalidParentHash)
}

func Test_Blockchain_InvalidHeight(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block := newTestBlock(t, bc.genesisBlock.HeaderHash, 0, db, 1, 3)

	err := bc.ValidateBlock(block)
	assert.Equal(t, err, ErrBlockInvalidHeight)
}
