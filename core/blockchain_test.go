/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/miner/pow"
)

type testAccount struct {
	addr    common.Address
	privKey *ecdsa.PrivateKey
	data    state.Account
}

var testGenesisAccounts = []*testAccount{
	newTestAccount(big.NewInt(100), 0),
	newTestAccount(big.NewInt(100), 0),
	newTestAccount(big.NewInt(100), 0),
}

func newTestAccount(amount *big.Int, nonce uint64) *testAccount {
	addr, privKey, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(err)
	}

	return &testAccount{
		addr:    *addr,
		privKey: privKey,
		data: state.Account{
			Amount: amount,
			Nonce:  nonce,
		},
	}
}

func newTestGenesis() *Genesis {
	accounts := make(map[common.Address]*big.Int)
	for _, account := range testGenesisAccounts {
		accounts[account.addr] = account.data.Amount
	}

	return GetGenesis(GenesisInfo{accounts, 1, 0})
}

func newTestBlockchain(db database.Database) *Blockchain {
	bcStore := store.NewBlockchainDatabase(db)

	genesis := newTestGenesis()
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	}

	bc, err := NewBlockchain(bcStore, db)
	if err != nil {
		panic(err)
	}

	return bc
}

func newTestBlockTx(genesisAccountIndex int, amount, nonce uint64) *types.Transaction {
	fromAccount := testGenesisAccounts[genesisAccountIndex]
	toAddress := crypto.MustGenerateRandomAddress()

	tx, _ := types.NewTransaction(fromAccount.addr, *toAddress, new(big.Int).SetUint64(amount), big.NewInt(0), nonce)
	tx.Sign(fromAccount.privKey)

	return tx
}

func newTestBlock(bc *Blockchain, parentHash common.Hash, blockHeight, txNum, startNonce uint64) *types.Block {
	minerAccount := newTestAccount(pow.GetReward(blockHeight), 0)
	rewardTx, _ := types.NewRewardTransaction(minerAccount.addr, minerAccount.data.Amount, uint64(1))

	txs := []*types.Transaction{rewardTx}
	for i := uint64(0); i < txNum; i++ {
		txs = append(txs, newTestBlockTx(0, 1, startNonce+i))
	}

	header := &types.BlockHeader{
		PreviousBlockHash: parentHash,
		Creator:           minerAccount.addr,
		StateHash:         common.EmptyHash,
		TxHash:            types.MerkleRootHash(txs),
		Height:            blockHeight,
		Difficulty:        big.NewInt(1),
		CreateTimestamp:   big.NewInt(1),
		Nonce:             10,
		ExtraData:         make([]byte, 0),
	}

	stateRootHash := common.EmptyHash
	receiptsRootHash := common.EmptyHash
	parentBlock, err := bc.bcStore.GetBlock(parentHash)
	if err == nil {
		statedb, err := state.NewStatedb(parentBlock.Header.StateHash, bc.accountStateDB)
		if err != nil {
			panic(err)
		}

		var receipts []*types.Receipt
		if receipts, err = bc.updateStateDB(statedb, rewardTx, txs[1:], header); err != nil {
			panic(err)
		}

		if stateRootHash, err = statedb.Commit(nil); err != nil {
			panic(err)
		}

		receiptsRootHash = types.ReceiptMerkleRootHash(receipts)
	}

	header.StateHash = stateRootHash
	header.ReceiptHash = receiptsRootHash

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

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	newBlock.HeaderHash = common.EmptyHash

	assert.Equal(t, bc.WriteBlock(newBlock), ErrBlockHashMismatch)
}

func Test_Blockchain_WriteBlock_TxRootHashChanged(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	newBlock.Header.TxHash = common.EmptyHash
	newBlock.HeaderHash = newBlock.Header.Hash()

	assert.Equal(t, bc.WriteBlock(newBlock), ErrBlockTxsHashMismatch)
}

func Test_Blockchain_WriteBlock_InvalidHeight(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	newBlock.Header.Height = 10
	newBlock.HeaderHash = newBlock.Header.Hash()

	assert.Equal(t, bc.WriteBlock(newBlock), ErrBlockInvalidHeight)
}

func Test_Blockchain_WriteBlock_InvalidExtraData(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	newBlock.Header.ExtraData = []byte("test extra data")
	newBlock.HeaderHash = newBlock.Header.Hash()

	assert.Equal(t, bc.WriteBlock(newBlock), ErrBlockExtraDataNotEmpty)
}

func Test_Blockchain_WriteBlock_ValidBlock(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	assert.Equal(t, bc.WriteBlock(newBlock), error(nil))

	currentBlock := bc.CurrentBlock()
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

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)

	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, error(nil))

	currentBlock := bc.CurrentBlock()
	assert.Equal(t, currentBlock, newBlock)

	err = bc.WriteBlock(newBlock)
	assert.Equal(t, err, ErrBlockAlreadyExists)
}

func Test_Blockchain_WriteBlock_InsertTwoBlocks(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block1 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	err := bc.WriteBlock(block1)
	assert.Equal(t, err, error(nil))

	currentBlock := bc.CurrentBlock()
	assert.Equal(t, currentBlock, block1)

	block2 := newTestBlock(bc, block1.HeaderHash, 2, 3, 3)
	err = bc.WriteBlock(block2)
	assert.Equal(t, err, error(nil))

	currentBlock = bc.CurrentBlock()
	assert.Equal(t, currentBlock, block2)
}

func Test_Blockchain_BlockFork(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block1 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	err := bc.WriteBlock(block1)
	assert.Equal(t, err, error(nil))

	currentBlock := bc.CurrentBlock()
	assert.Equal(t, currentBlock, block1)
	assert.Equal(t, bc.blockLeaves.Count(), 1)

	block2 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	err = bc.WriteBlock(block2)
	assert.Equal(t, err, error(nil))

	assert.Equal(t, bc.blockLeaves.Count(), 2)
}

func Test_BlockChain_InvalidParent(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block := newTestBlock(bc, common.EmptyHash, 1, 3, 0)
	assert.Equal(t, bc.WriteBlock(block), ErrBlockInvalidParentHash)
}

func Test_Blockchain_InvalidHeight(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block := newTestBlock(bc, bc.genesisBlock.HeaderHash, 0, 3, 0)
	assert.Equal(t, bc.WriteBlock(block), ErrBlockInvalidHeight)
}

func Test_Blockchain_UpdateCanocialHash(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)
	assertCanonicalHash(t, bc, 0, bc.genesisBlock.HeaderHash)

	// genesis <- block11
	block11 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	assert.Equal(t, bc.WriteBlock(block11), error(nil))
	assertCanonicalHash(t, bc, 1, block11.HeaderHash)

	// genesis <- block11 <- block12
	block12 := newTestBlock(bc, block11.HeaderHash, 2, 3, 3)
	assert.Equal(t, bc.WriteBlock(block12), error(nil))
	assertCanonicalHash(t, bc, 2, block12.HeaderHash)

	// genesis <- block11 <- block12 (canonical)
	//         <- block21
	block21 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	assert.Equal(t, bc.WriteBlock(block21), error(nil))
	assertCanonicalHash(t, bc, 1, block11.HeaderHash)
	assertCanonicalHash(t, bc, 2, block12.HeaderHash)

	// genesis <- block11 <- block12 (canonical)
	//         <- block21 <- block22
	block22 := newTestBlock(bc, block21.HeaderHash, 2, 3, 3)
	assert.Equal(t, bc.WriteBlock(block22), error(nil))
	assertCanonicalHash(t, bc, 1, block11.HeaderHash)
	assertCanonicalHash(t, bc, 2, block12.HeaderHash)

	// genesis <- block11 <- block12
	//         <- block21 <- block22 <- block23 (canonical)
	block23 := newTestBlock(bc, block22.HeaderHash, 3, 3, 6)
	assert.Equal(t, bc.WriteBlock(block23), error(nil))
	assertCanonicalHash(t, bc, 1, block21.HeaderHash)
	assertCanonicalHash(t, bc, 2, block22.HeaderHash)
	assertCanonicalHash(t, bc, 3, block23.HeaderHash)
}

func assertCanonicalHash(t *testing.T, bc *Blockchain, height uint64, expectedHash common.Hash) {
	hash, err := bc.bcStore.GetBlockHash(height)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, hash, expectedHash)
}

func Test_Blockchain_Shard(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	bcStore := store.NewBlockchainDatabase(db)
	genesis := GetGenesis(GenesisInfo{nil, 1, 8})
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	}

	bc, err := NewBlockchain(bcStore, db)
	if err != nil {
		panic(err)
	}

	shardNum, err := bc.GetShardNumber()
	assert.Equal(t, err, nil)
	assert.Equal(t, shardNum, uint(8))
}
