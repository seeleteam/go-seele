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
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
)

func randomAccount(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	privKey, keyErr := crypto.GenerateKey()
	if keyErr != nil {
		t.Fatalf("Failed to generate ECDSA private key, error = %s", keyErr.Error())
	}

	hexAddress := crypto.PubkeyToString(&privKey.PublicKey)

	return privKey, common.HexMustToAddres(hexAddress)
}

func newTestPoolTx(t *testing.T, amount int64, nonce uint64) *poolItem {
	return newTestPoolTxWithNonce(t, amount, nonce, 1)
}

func newTestPoolTxWithNonce(t *testing.T, amount int64, nonce uint64, price int64) *poolItem {
	fromPrivKey, fromAddress := randomAccount(t)

	return newTestPoolEx(t, fromPrivKey, fromAddress, amount, nonce, price)
}

func newTestPoolEx(t *testing.T, fromPrivKey *ecdsa.PrivateKey, fromAddress common.Address, amount int64, nonce uint64, price int64) *poolItem {
	_, toAddress := randomAccount(t)

	tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(price), nonce)
	tx.Sign(fromPrivKey)

	return newPooledItem(tx)
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

func newTestPool(config *TransactionPoolConfig) (*Pool, *mockBlockchain) {
	chain := newMockBlockchain()
	pool := &Pool{
		capacity:          config.Capacity,
		chain:             chain,
		hashToTxMap:       make(map[common.Hash]*poolItem),
		pendingQueue:      newPendingQueue(),
		processingObjects: make(map[common.Hash]struct{}),
		log:               log.GetLogger("test"),
	}

	return pool, chain
}

func (chain mockBlockchain) addAccount(addr common.Address, balance, nonce uint64) {
	chain.statedb.CreateAccount(addr)
	chain.statedb.SetBalance(addr, new(big.Int).SetUint64(balance))
	chain.statedb.SetNonce(addr, nonce)
}

func Test_TransactionPool_Add_ValidTx(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Account(), 50000, 100)

	err := pool.AddObject(poolTx.poolObject)

	assert.Equal(t, err, error(nil))
	assert.Equal(t, len(pool.hashToTxMap), 1)
}

func Test_TransactionPool_Add_DuplicateTx(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Account(), 50000, 100)

	err := pool.AddObject(poolTx.poolObject)
	assert.Equal(t, err, error(nil))

	err = pool.AddObject(poolTx.poolObject)
	assert.Equal(t, err, errObjectHashExists)
}

func Test_TransactionPool_Add_PoolFull(t *testing.T) {
	config := DefaultTxPoolConfig()
	config.Capacity = 1
	pool, chain := newTestPool(config)
	defer chain.dispose()

	// tx with price 5
	poolTx1 := newTestPoolTxWithNonce(t, 10, 100, 5)
	chain.addAccount(poolTx1.Account(), 5000000, 100)
	assert.Nil(t, pool.AddObject(poolTx1.poolObject))

	// failed to add tx with same price
	poolTx2 := newTestPoolTxWithNonce(t, 10, 100, 5)
	chain.addAccount(poolTx2.Account(), 5000000, 100)
	assert.Equal(t, errObjectPoolFull, pool.AddObject(poolTx2.poolObject))

	// succeed to add tx with higher price
	poolTx3 := newTestPoolTxWithNonce(t, 10, 100, 6)
	chain.addAccount(poolTx3.Account(), 5000000, 100)
	assert.Nil(t, pool.AddObject(poolTx3.poolObject))
	assert.Nil(t, pool.hashToTxMap[poolTx1.GetHash()])
	assert.Equal(t, poolTx3.poolObject, pool.hashToTxMap[poolTx3.GetHash()].poolObject)
}

func Test_TransactionPool_Add_TxNonceUsed(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	fromPrivKey, fromAddress := randomAccount(t)
	var nonce uint64 = 100
	poolTx := newTestPoolEx(t, fromPrivKey, fromAddress, 10, nonce, 1)
	chain.addAccount(poolTx.Account(), 50000, 10)

	err := pool.AddObject(poolTx.poolObject)
	assert.Equal(t, err, error(nil))

	poolTx = newTestPoolEx(t, fromPrivKey, fromAddress, 10, nonce, 1)
	err = pool.AddObject(poolTx.poolObject)
	assert.Equal(t, err, errObjectNonceUsed)
}

func Test_TransactionPool_GetTransaction(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Account(), 50000, 100)

	pool.AddObject(poolTx.poolObject)

	assert.Equal(t, pool.GetObject(poolTx.GetHash()), poolTx.poolObject)
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
	chain.addAccount(poolTx.Account(), 50000, 100)

	err := pool.AddObject(poolTx.poolObject)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, pool.pendingQueue.count(), 1)

	pool.RemoveOject(poolTx.GetHash())
	assert.Equal(t, pool.pendingQueue.count(), 0)
}

func Test_TransactionPool_GetPendingTxCount(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	assert.Equal(t, pool.GetPendingObjectCount(), 0)

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Account(), 50000, 100)

	err := pool.AddObject(poolTx.poolObject)
	assert.Equal(t, err, nil)
	assert.Equal(t, pool.GetPendingObjectCount(), 1)

	txs, size := pool.GetProcessableObjects(BlockByteLimit)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, size, types.TransactionPreSize)

	assert.Equal(t, pool.GetPendingObjectCount(), 0)
}

func Test_TransactionPool_GetTransactions(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Account(), 500000, 100)

	pool.AddObject(poolTx.poolObject)

	txs := pool.GetObjects(true, false)
	assert.Equal(t, len(txs), 0)

	txs = pool.GetObjects(false, true)
	assert.Equal(t, len(txs), 1)

	txs, size := pool.GetProcessableObjects(BlockByteLimit)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, size, types.TransactionPreSize)

	txs = pool.GetObjects(true, false)
	assert.Equal(t, len(txs), 1)

	txs = pool.GetObjects(false, true)
	assert.Equal(t, len(txs), 0)
}

func newTxs(t *testing.T, amount, price, nonce, number int64, chain *mockBlockchain) []poolObject {
	var txs []poolObject

	for i := int64(0); i < number; i++ {
		fromPrivKey, fromAddress := randomAccount(t)
		_, toAddress := randomAccount(t)
		tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(price), uint64(nonce))
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
	pool.addObjects(txs)

	txs, size := pool.GetProcessableObjects(0)
	assert.Equal(t, len(txs), 0)
	assert.Equal(t, size, 0)

	txs, size = pool.GetProcessableObjects(types.TransactionPreSize)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, size, types.TransactionPreSize)

	txs, size = pool.GetProcessableObjects(types.TransactionPreSize * 2)
	assert.Equal(t, len(txs), 2)
	assert.Equal(t, size, types.TransactionPreSize*2)

	txs, size = pool.GetProcessableObjects(types.TransactionPreSize * 10)
	assert.Equal(t, len(txs), 7)
	assert.Equal(t, size, types.TransactionPreSize*7)

	txs, size = pool.GetProcessableObjects(types.TransactionPreSize * 10)
	assert.Equal(t, len(txs), 0)
	assert.Equal(t, size, 0)
}
