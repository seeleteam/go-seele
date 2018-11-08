/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/pow"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
	leveldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
)

type testAccount struct {
	addr    common.Address
	privKey *ecdsa.PrivateKey
	amount  *big.Int
	nonce   uint64
}

// genesis account with enough balance (100K seele) for benchmark test
var genesisAccount = newTestAccount(new(big.Int).Mul(big.NewInt(100000), common.SeeleToFan), 0, 1)

func newTestAccount(amount *big.Int, nonce uint64, shard uint) *testAccount {
	addr, privKey := crypto.MustGenerateShardKeyPair(shard)

	return &testAccount{
		addr:    *addr,
		privKey: privKey,
		amount:  new(big.Int).Set(amount),
		nonce:   nonce,
	}
}

func newTestGenesis() *Genesis {
	accounts := map[common.Address]*big.Int{
		genesisAccount.addr: genesisAccount.amount,
	}

	return GetGenesis(GenesisInfo{accounts, 1, 0, big.NewInt(0)})
}

func newTestBlockchain(db database.Database) *Blockchain {
	bcStore := store.NewCachedStore(store.NewBlockchainDatabase(db))

	genesis := newTestGenesis()
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	}

	bc, err := NewBlockchain(bcStore, db, "", pow.NewEngine(1), nil)
	if err != nil {
		panic(err)
	}

	return bc
}

func newTestBlockTx(amount, price, nonce uint64) *types.Transaction {
	toAddress := crypto.MustGenerateShardAddress(genesisAccount.addr.Shard())

	tx, _ := types.NewTransaction(genesisAccount.addr, *toAddress, new(big.Int).SetUint64(amount), new(big.Int).SetUint64(price), nonce)
	tx.Sign(genesisAccount.privKey)

	return tx
}

func newTestBlock(bc *Blockchain, parentHash common.Hash, blockHeight, startNonce uint64, size int) *types.Block {
	minerAccount := newTestAccount(consensus.GetReward(blockHeight), 0, 1)
	rewardTx, _ := types.NewRewardTransaction(minerAccount.addr, minerAccount.amount, uint64(1))

	txs := []*types.Transaction{rewardTx}
	totalSize := rewardTx.Size()
	for i := uint64(0); ; i++ {
		tx := newTestBlockTx(1, 1, startNonce+i)
		tmp := tx.Size() + totalSize
		if tmp > size {
			break
		}

		txs = append(txs, tx)
		totalSize = tmp

	}

	header := &types.BlockHeader{
		PreviousBlockHash: parentHash,
		Creator:           minerAccount.addr,
		StateHash:         common.EmptyHash,
		TxHash:            types.MerkleRootHash(txs),
		TxDebtHash:        types.DebtMerkleRootHash(types.NewDebts(txs)),
		DebtHash:          common.EmptyHash,
		Height:            blockHeight,
		Difficulty:        big.NewInt(1),
		CreateTimestamp:   big.NewInt(1),
		Witness:           make([]byte, 0),
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

		if stateRootHash, err = statedb.Hash(); err != nil {
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
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	newBlock.HeaderHash = common.EmptyHash
	assert.True(t, errors.IsOrContains(bc.WriteBlock(newBlock), types.ErrBlockHashMismatch))
}

func Test_Blockchain_WriteBlock_TxRootHashChanged(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	newBlock.Header.TxHash = common.EmptyHash
	newBlock.HeaderHash = newBlock.Header.Hash()

	assert.True(t, errors.IsOrContains(bc.WriteBlock(newBlock), types.ErrBlockTxsHashMismatch))
}

func Test_Blockchain_WriteBlock_InvalidHeight(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	newBlock.Header.Height = 10
	newBlock.HeaderHash = newBlock.Header.Hash()

	assert.True(t, errors.IsOrContains(bc.WriteBlock(newBlock), consensus.ErrBlockInvalidHeight))
}

func Test_Blockchain_WriteBlock_InvalidExtraData(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	newBlock.Header.ExtraData = []byte("test extra data")
	newBlock.HeaderHash = newBlock.Header.Hash()

	assert.True(t, errors.IsOrContains(bc.WriteBlock(newBlock), ErrBlockExtraDataNotEmpty))
}

func Test_Blockchain_WriteBlock_ValidBlock(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
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
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	newBlock := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)

	err := bc.WriteBlock(newBlock)
	assert.Equal(t, err, error(nil))

	currentBlock := bc.CurrentBlock()
	assert.Equal(t, currentBlock, newBlock)

	err = bc.WriteBlock(newBlock)
	assert.True(t, errors.IsOrContains(err, ErrBlockAlreadyExists))
}

func Test_Blockchain_WriteBlock_InsertTwoBlocks(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
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
	db, dispose := leveldb.NewTestDatabase()
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
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block := newTestBlock(bc, common.EmptyHash, 1, 3, 0)
	assert.True(t, errors.IsOrContains(bc.WriteBlock(block), consensus.ErrBlockInvalidParentHash))
}

func Test_Blockchain_InvalidHeight(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	block := newTestBlock(bc, bc.genesisBlock.HeaderHash, 0, 3, 0)
	assert.True(t, errors.IsOrContains(bc.WriteBlock(block), consensus.ErrBlockInvalidHeight))
}

func Test_Blockchain_UpdateCanocialHash(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)
	assertCanonicalHash(t, bc, 0, bc.genesisBlock.HeaderHash)

	// genesis <- block11
	block11 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	assert.Equal(t, bc.WriteBlock(block11), error(nil))
	assertCanonicalHash(t, bc, 1, block11.HeaderHash)
	assertTxDebtIndex(t, bc, true, block11)

	// genesis <- block11 <- block12
	block12 := newTestBlock(bc, block11.HeaderHash, 2, 3, 3)
	assert.Equal(t, bc.WriteBlock(block12), error(nil))
	assertCanonicalHash(t, bc, 2, block12.HeaderHash)
	assertTxDebtIndex(t, bc, true, block11, block12)

	// genesis <- block11 <- block12 (canonical)
	//         <- block21
	block21 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	assert.Equal(t, bc.WriteBlock(block21), error(nil))
	assertCanonicalHash(t, bc, 1, block11.HeaderHash)
	assertCanonicalHash(t, bc, 2, block12.HeaderHash)
	assertTxDebtIndex(t, bc, true, block11, block12)
	assertTxDebtIndex(t, bc, false, block21)

	// genesis <- block11 <- block12 (canonical)
	//         <- block21 <- block22
	block22 := newTestBlock(bc, block21.HeaderHash, 2, 3, 3)
	assert.Equal(t, bc.WriteBlock(block22), error(nil))
	assertCanonicalHash(t, bc, 1, block11.HeaderHash)
	assertCanonicalHash(t, bc, 2, block12.HeaderHash)
	assertTxDebtIndex(t, bc, true, block11, block12)
	assertTxDebtIndex(t, bc, false, block21, block22)

	// genesis <- block11 <- block12
	//         <- block21 <- block22 <- block23 (canonical)
	block23 := newTestBlock(bc, block22.HeaderHash, 3, 3, 6)
	assert.Equal(t, bc.WriteBlock(block23), error(nil))
	assertCanonicalHash(t, bc, 1, block21.HeaderHash)
	assertCanonicalHash(t, bc, 2, block22.HeaderHash)
	assertCanonicalHash(t, bc, 3, block23.HeaderHash)
	assertTxDebtIndex(t, bc, false, block11, block12)
	assertTxDebtIndex(t, bc, true, block21, block22, block23)
}

func assertCanonicalHash(t *testing.T, bc *Blockchain, height uint64, expectedHash common.Hash) {
	hash, err := bc.bcStore.GetBlockHash(height)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, hash, expectedHash)
}

func assertTxDebtIndex(t *testing.T, bc *Blockchain, exists bool, blocks ...*types.Block) {
	for _, block := range blocks {
		for i, tx := range block.Transactions {
			idx, err := bc.bcStore.GetTxIndex(tx.Hash)
			if exists {
				assert.Equal(t, block.HeaderHash, idx.BlockHash)
				assert.Equal(t, uint(i), idx.Index)
			} else {
				assert.Equal(t, leveldbErrors.ErrNotFound, err)
			}
		}

		for i, debt := range block.Debts {
			idx, err := bc.bcStore.GetDebtIndex(debt.Hash)
			if exists {
				assert.Equal(t, block.HeaderHash, idx.BlockHash)
				assert.Equal(t, uint(i), idx.Index)
			} else {
				assert.Equal(t, leveldbErrors.ErrNotFound, err)
			}
		}
	}
}

func Test_Blockchain_Shard(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bcStore := store.NewBlockchainDatabase(db)
	genesis := GetGenesis(GenesisInfo{nil, 1, 8, big.NewInt(0)})
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	}

	bc, err := NewBlockchain(bcStore, db, "", pow.NewEngine(1), nil)
	if err != nil {
		panic(err)
	}

	shardNum, err := bc.GetShardNumber()
	assert.Equal(t, err, nil)
	assert.Equal(t, shardNum, uint(8))
}

func Test_Blockchain_ApplyTransaction(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)

	// prepare tx to apply
	amount := uint64(3456)
	price := uint64(2)
	tx := newTestBlockTx(amount, price, 0)
	block := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 1, 0)
	coinbase := block.Header.Creator
	statedb, err := bc.GetCurrentState()
	assert.Equal(t, err, nil)
	statedb.CreateAccount(coinbase)
	statedb.SetBalance(coinbase, big.NewInt(50))

	// check before applying tx
	assert.Equal(t, statedb.GetBalance(tx.Data.From), genesisAccount.amount)
	assert.Equal(t, statedb.GetBalance(tx.Data.To), big.NewInt(-1))
	assert.Equal(t, statedb.GetBalance(coinbase), big.NewInt(50))

	// apply tx
	_, err = bc.ApplyTransaction(tx, 1, coinbase, statedb, block.Header)
	assert.Equal(t, err, nil)

	// check after applying tx
	used := new(big.Int).SetUint64(amount + price*types.TransferAmountIntrinsicGas)
	newBalance := new(big.Int).Sub(genesisAccount.amount, used)
	assert.Equal(t, statedb.GetBalance(tx.Data.From), newBalance)
	assert.Equal(t, statedb.GetBalance(tx.Data.To), new(big.Int).SetUint64(amount))
	assert.Equal(t, statedb.GetBalance(coinbase), new(big.Int).SetUint64(50+price*types.TransferAmountIntrinsicGas))
}

func Benchmark_Blockchain_WriteBlock(b *testing.B) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)
	preBlock := bc.genesisBlock

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		block := newTestBlock(bc, preBlock.HeaderHash, preBlock.Header.Height+1, 0, BlockByteLimit)
		b.StartTimer()
		if err := bc.WriteBlock(block); err != nil {
			b.Fatalf("failed to write block, %v", err.Error())
		}
		preBlock = block
	}
}

func Benchmark_Blockchain_ValidateTxs(b *testing.B) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)
	preBlock := bc.genesisBlock
	block := newTestBlock(bc, preBlock.HeaderHash, preBlock.Header.Height+1, 0, BlockByteLimit)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		types.BatchValidateTxs(block.Transactions[1:])
	}
}
