/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/log"
)

func randomAccount(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	privKey, keyErr := crypto.GenerateKey()
	if keyErr != nil {
		t.Fatalf("Failed to generate ECDSA private key, error = %s", keyErr.Error())
	}

	hexAddress := crypto.PubkeyToString(&privKey.PublicKey)

	return privKey, common.HexMustToAddres(hexAddress)
}

func newTestPoolTx(t *testing.T, amount int64, nonce uint64) *pooledTx {
	return newTestPoolTxWithNonce(t, amount, nonce, 1)
}

func newTestPoolTxWithNonce(t *testing.T, amount int64, nonce uint64, fee int64) *pooledTx {
	fromPrivKey, fromAddress := randomAccount(t)

	return newTestPoolEx(t, fromPrivKey, fromAddress, amount, nonce, 1)
}

func newTestPoolEx(t *testing.T, fromPrivKey *ecdsa.PrivateKey, fromAddress common.Address, amount int64, nonce uint64, fee int64) *pooledTx {
	_, toAddress := randomAccount(t)

	tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(fee), nonce)
	tx.Sign(fromPrivKey)

	return newPooledTx(tx)
}

type mockBlockchain struct {
	statedb    *state.Statedb
	chainStore store.BlockchainStore
	dispose    func()
}

func newMockBlockchain() *mockBlockchain {
	statedb, err := state.NewStatedb(common.EmptyHash, nil)
	if err != nil {
		panic(err)
	}

	db, dispose := leveldb.NewTestDatabase()
	chainStore := store.NewBlockchainDatabase(db)
	return &mockBlockchain{statedb, chainStore, dispose}
}

func (chain mockBlockchain) GetCurrentState() (*state.Statedb, error) {
	return chain.statedb, nil
}

func (chain mockBlockchain) GetStore() store.BlockchainStore {
	return chain.chainStore
}

func newTestPool(config *TransactionPoolConfig) (*TransactionPool, *mockBlockchain) {
	chain := newMockBlockchain()
	pool := &TransactionPool{
		config:        *config,
		chain:         chain,
		hashToTxMap:   make(map[common.Hash]*pooledTx),
		pendingQueue:  newPendingQueue(),
		processingTxs: make(map[common.Hash]struct{}),
		lastHeader:    common.EmptyHash,
		log:           log.GetLogger("test", true),
	}

	return pool, chain
}

func (chain mockBlockchain) addAccount(addr common.Address, balance, nonce uint64) {
	stateObj := chain.statedb.GetOrNewStateObject(addr)
	stateObj.SetAmount(new(big.Int).SetUint64(balance))
	stateObj.SetNonce(nonce)
}

func Test_TransactionPool_Add_ValidTx(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.Transaction)

	assert.Equal(t, err, error(nil))
	assert.Equal(t, len(pool.hashToTxMap), 1)
}

func Test_TransactionPool_Add_InvalidTx(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Data.From, 20, 100)

	// Change the amount in tx.
	poolTx.Data.Amount.SetInt64(20)
	err := pool.AddTransaction(poolTx.Transaction)

	if err == nil {
		t.Fatal("The error is nil when add invalid tx to pool.")
	}

	// add nil tx
	err = pool.AddTransaction(nil)
	assert.Equal(t, err, error(nil))
}

func Test_TransactionPool_Add_DuplicateTx(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.Transaction)
	assert.Equal(t, err, error(nil))

	err = pool.AddTransaction(poolTx.Transaction)
	assert.Equal(t, err, errTxHashExists)
}

func Test_TransactionPool_Add_PoolFull(t *testing.T) {
	config := DefaultTxPoolConfig()
	config.Capacity = 1
	pool, chain := newTestPool(config)
	defer chain.dispose()

	poolTx1 := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx1.Data.From, 20, 100)
	poolTx2 := newTestPoolTx(t, 19, 101)
	chain.addAccount(poolTx2.Data.From, 20, 101)

	err := pool.AddTransaction(poolTx1.Transaction)
	assert.Equal(t, err, error(nil))

	err = pool.AddTransaction(poolTx2.Transaction)
	assert.Equal(t, err, errTxPoolFull)
}

func Test_TransactionPool_Add_TxNonceUsed(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	fromPrivKey, fromAddress := randomAccount(t)
	var nonce uint64 = 100
	poolTx := newTestPoolEx(t, fromPrivKey, fromAddress, 10, nonce, 10)
	chain.addAccount(poolTx.Data.From, 20, 10)

	err := pool.AddTransaction(poolTx.Transaction)
	assert.Equal(t, err, error(nil))

	poolTx = newTestPoolEx(t, fromPrivKey, fromAddress, 10, nonce, 8)
	err = pool.AddTransaction(poolTx.Transaction)
	assert.Equal(t, err, errTxNonceUsed)
}

func Test_TransactionPool_GetTransaction(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Data.From, 20, 100)

	pool.AddTransaction(poolTx.Transaction)

	assert.Equal(t, pool.GetTransaction(poolTx.Hash), poolTx.Transaction)
}

func newTestAccountTxs(t *testing.T, amounts []int64, nonces []uint64) (common.Address, []*types.Transaction) {
	if len(amounts) != len(nonces) || len(amounts) == 0 {
		t.Fatal()
	}

	fromPrivKey, fromAddress := randomAccount(t)
	txs := make([]*types.Transaction, 0, len(amounts))

	for i, amount := range amounts {
		_, toAddress := randomAccount(t)

		tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(1), nonces[i])
		tx.Sign(fromPrivKey)

		txs = append(txs, tx)
	}

	return fromAddress, txs
}

func Test_TransactionPool_Remove(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.Transaction)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, pool.pendingQueue.count(), 1)

	pool.RemoveTransaction(poolTx.Hash)
	assert.Equal(t, pool.pendingQueue.count(), 0)
}

func Test_GetRejectTransacton(t *testing.T) {
	log := log.GetLogger("test", true)
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)
	b1 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 0, 4*types.TransactionPreSize)
	bc.WriteBlock(b1)

	b2 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 0, 3*types.TransactionPreSize)
	bc.WriteBlock(b2)

	reinject := getReinjectTransaction(bc.GetStore(), b1.HeaderHash, b2.HeaderHash, log)

	assert.Equal(t, len(reinject), 2)
	txs := make(map[common.Hash]*types.Transaction)
	for _, tx := range reinject {
		txs[tx.Hash] = tx
	}

	_, ok := txs[b2.Transactions[1].Hash]
	assert.Equal(t, ok, true, "1")

	_, ok = txs[b2.Transactions[2].Hash]
	assert.Equal(t, ok, true, "2")
}

func Test_TransactionPool_RemoveTransactions(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.Transaction)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, pool.pendingQueue.count(), 1)

	for _, ptx := range pool.hashToTxMap {
		ptx.timestamp = ptx.timestamp.Add(-10 * time.Second)
	}

	pool.removeTransactions()
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, pool.pendingQueue.count(), 1)

	for _, ptx := range pool.hashToTxMap {
		ptx.timestamp = ptx.timestamp.Add(-overTimeInterval)
	}

	pool.removeTransactions()
	assert.Equal(t, len(pool.hashToTxMap), 0)
	assert.Equal(t, pool.pendingQueue.count(), 0)
}

func Test_TransactionPool_GetPendingTxCount(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	assert.Equal(t, pool.GetPendingTxCount(), 0)

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.Transaction)
	assert.Equal(t, err, nil)
	assert.Equal(t, pool.GetPendingTxCount(), 1)

	txs, size := pool.GetProcessableTransactions(BlockByteLimit)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, size, types.TransactionPreSize)

	assert.Equal(t, pool.GetPendingTxCount(), 0)
}

func Test_TransactionPool_GetTransactions(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Data.From, 20, 100)

	pool.AddTransaction(poolTx.Transaction)

	txs := pool.GetTransactions(true, false)
	assert.Equal(t, len(txs), 0)

	txs = pool.GetTransactions(false, true)
	assert.Equal(t, len(txs), 1)

	txs, size := pool.GetProcessableTransactions(BlockByteLimit)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, size, types.TransactionPreSize)

	txs = pool.GetTransactions(true, false)
	assert.Equal(t, len(txs), 1)

	txs = pool.GetTransactions(false, true)
	assert.Equal(t, len(txs), 0)
}

func newTxs(t *testing.T, amount, fee, nonce, number int64, chain *mockBlockchain) []*types.Transaction {
	var txs []*types.Transaction

	for i := int64(0); i < number; i++ {
		fromPrivKey, fromAddress := randomAccount(t)
		_, toAddress := randomAccount(t)
		tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(fee), uint64(nonce))
		tx.Sign(fromPrivKey)
		chain.addAccount(fromAddress, 1000000000, 1)
		txs = append(txs, tx)
	}
	return txs
}

func Test_TransactionPool_GetProcessableTransactions(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	txs := newTxs(t, 10, 10, 1, 10, chain)
	pool.addTransactions(txs)

	txs, size := pool.GetProcessableTransactions(0)
	assert.Equal(t, len(txs), 0)
	assert.Equal(t, size, 0)

	txs, size = pool.GetProcessableTransactions(types.TransactionPreSize)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, size, types.TransactionPreSize)

	txs, size = pool.GetProcessableTransactions(types.TransactionPreSize * 2)
	assert.Equal(t, len(txs), 2)
	assert.Equal(t, size, types.TransactionPreSize*2)

	txs, size = pool.GetProcessableTransactions(types.TransactionPreSize * 10)
	assert.Equal(t, len(txs), 7)
	assert.Equal(t, size, types.TransactionPreSize*7)

	txs, size = pool.GetProcessableTransactions(types.TransactionPreSize * 10)
	assert.Equal(t, len(txs), 0)
	assert.Equal(t, size, 0)
}
