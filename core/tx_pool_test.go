/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func newTestTransactionPool(config *TransactionPoolConfig) (*TransactionPool, *mockBlockchain) {
	chain := newMockBlockchain()
	pool := NewTransactionPool(*config, chain)

	return pool, chain
}

func Test_TransactionPool_Add_InvalidTx(t *testing.T) {
	pool, chain := newTestTransactionPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 30, 100)
	chain.addAccount(poolTx.FromAccount(), 20, 100)

	// Change the amount in tx.
	err := pool.AddTransaction(poolTx.poolObject.(*types.Transaction))

	if err == nil {
		t.Fatal("The error is nil when add invalid tx to pool.")
	}

	// add nil tx
	err = pool.AddTransaction(nil)
	assert.Equal(t, err, error(nil))
}

func Test_TransactionPool_RemoveTransactions(t *testing.T) {
	pool, chain := newTestTransactionPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.FromAccount(), 500000, 100)

	err := pool.addObject(poolTx.poolObject)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, pool.pendingQueue.count(), 1)

	for _, ptx := range pool.hashToTxMap {
		ptx.timestamp = ptx.timestamp.Add(-10 * time.Second)
	}

	pool.removeObjects()
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, pool.pendingQueue.count(), 1)

	for _, ptx := range pool.hashToTxMap {
		ptx.timestamp = ptx.timestamp.Add(-transactionTimeoutDuration)
	}

	pool.removeObjects()
	assert.Equal(t, len(pool.hashToTxMap), 0)
	assert.Equal(t, pool.pendingQueue.count(), 0)
}

func Test_GetReinjectTransaction(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)
	pool := NewTransactionPool(*DefaultTxPoolConfig(), bc)

	b1 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 0, 4*types.TransactionPreSize)
	bc.WriteBlock(b1)

	b2 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 0, 3*types.TransactionPreSize)
	bc.WriteBlock(b2)

	reinject := pool.getReinjectObject(b1.HeaderHash, b2.HeaderHash)

	assert.Equal(t, len(reinject), 2)
	txs := make(map[common.Hash]poolObject)
	for _, tx := range reinject {
		txs[tx.GetHash()] = tx
	}

	_, ok := txs[b2.Transactions[1].Hash]
	assert.Equal(t, ok, true, "1")

	_, ok = txs[b2.Transactions[2].Hash]
	assert.Equal(t, ok, true, "2")
}

func Test_TransactionPool_Add_ValidTx(t *testing.T) {
	pool, chain := newTestTransactionPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.FromAccount(), 50000, 100)

	err := pool.addObject(poolTx.poolObject)

	assert.Equal(t, err, error(nil))
	assert.Equal(t, len(pool.hashToTxMap), 1)
}

func Test_TransactionPool_Add_DuplicateTx(t *testing.T) {
	pool, chain := newTestTransactionPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.FromAccount(), 50000, 100)

	err := pool.addObject(poolTx.poolObject)
	assert.Equal(t, err, error(nil))

	err = pool.addObject(poolTx.poolObject)
	assert.Equal(t, err, errObjectHashExists)
}

func Test_TransactionPool_Add_PoolFull(t *testing.T) {
	config := DefaultTxPoolConfig()
	config.Capacity = 1
	pool, chain := newTestTransactionPool(config)
	defer chain.dispose()

	// tx with price 5
	poolTx1 := newTestPoolTxWithNonce(t, 10, 100, 5)
	chain.addAccount(poolTx1.FromAccount(), 5000000, 100)
	assert.Nil(t, pool.addObject(poolTx1.poolObject))

	// failed to add tx with same price
	poolTx2 := newTestPoolTxWithNonce(t, 10, 100, 5)
	chain.addAccount(poolTx2.FromAccount(), 5000000, 100)
	assert.Equal(t, errObjectPoolFull, pool.addObject(poolTx2.poolObject))

	// succeed to add tx with higher price
	poolTx3 := newTestPoolTxWithNonce(t, 10, 100, 6)
	chain.addAccount(poolTx3.FromAccount(), 5000000, 100)
	assert.Nil(t, pool.addObject(poolTx3.poolObject))
	assert.Nil(t, pool.hashToTxMap[poolTx1.GetHash()])
	assert.Equal(t, poolTx3.poolObject, pool.hashToTxMap[poolTx3.GetHash()].poolObject)
}

func Test_TransactionPool_Add_TxNonceUsed(t *testing.T) {
	pool, chain := newTestTransactionPool(DefaultTxPoolConfig())
	defer chain.dispose()

	fromPrivKey, fromAddress := randomAccount(t)
	var nonce uint64 = 100
	poolTx := newTestPoolEx(t, fromPrivKey, fromAddress, 10, nonce, 1)
	chain.addAccount(poolTx.FromAccount(), 50000, 10)

	err := pool.addObject(poolTx.poolObject)
	assert.Equal(t, err, error(nil))

	poolTx = newTestPoolEx(t, fromPrivKey, fromAddress, 10, nonce, 1)
	err = pool.addObject(poolTx.poolObject)
	assert.Equal(t, err, errObjectNonceUsed)
}

func Test_TransactionPool_GetTransaction(t *testing.T) {
	pool, chain := newTestTransactionPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.FromAccount(), 50000, 100)

	pool.addObject(poolTx.poolObject)

	assert.Equal(t, pool.GetObject(poolTx.GetHash()), poolTx.poolObject)
}

func Test_TransactionPool_Remove(t *testing.T) {
	pool, chain := newTestTransactionPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.FromAccount(), 50000, 100)

	err := pool.addObject(poolTx.poolObject)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, pool.pendingQueue.count(), 1)

	pool.removeOject(poolTx.GetHash())
	assert.Equal(t, pool.pendingQueue.count(), 0)
}

func Test_TransactionPool_GetPendingTxCount(t *testing.T) {
	pool, chain := newTestTransactionPool(DefaultTxPoolConfig())
	defer chain.dispose()

	assert.Equal(t, pool.GetPendingTxCount(), 0)

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.FromAccount(), 50000, 100)

	err := pool.addObject(poolTx.poolObject)
	assert.Equal(t, err, nil)
	assert.Equal(t, pool.GetPendingTxCount(), 1)

	txs, size := pool.getProcessableObjects(BlockByteLimit)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, size, types.TransactionPreSize)

	assert.Equal(t, pool.GetPendingTxCount(), 0)
}

func Test_TransactionPool_GetTransactions(t *testing.T) {
	pool, chain := newTestTransactionPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.FromAccount(), 500000, 100)

	pool.addObject(poolTx.poolObject)

	txs := pool.getObjects(true, false)
	assert.Equal(t, len(txs), 0)

	txs = pool.getObjects(false, true)
	assert.Equal(t, len(txs), 1)

	txs, size := pool.getProcessableObjects(BlockByteLimit)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, size, types.TransactionPreSize)

	txs = pool.getObjects(true, false)
	assert.Equal(t, len(txs), 1)

	txs = pool.getObjects(false, true)
	assert.Equal(t, len(txs), 0)
}

func Test_TransactionPool_GetProcessableTransactions(t *testing.T) {
	pool, chain := newTestTransactionPool(DefaultTxPoolConfig())
	defer chain.dispose()

	txs := newTxs(t, 10, 10, 1, 10, chain)
	pool.addObjectArray(txs)

	txs, size := pool.getProcessableObjects(0)
	assert.Equal(t, len(txs), 0)
	assert.Equal(t, size, 0)

	txs, size = pool.getProcessableObjects(types.TransactionPreSize)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, size, types.TransactionPreSize)

	txs, size = pool.getProcessableObjects(types.TransactionPreSize * 2)
	assert.Equal(t, len(txs), 2)
	assert.Equal(t, size, types.TransactionPreSize*2)

	txs, size = pool.getProcessableObjects(types.TransactionPreSize * 10)
	assert.Equal(t, len(txs), 7)
	assert.Equal(t, size, types.TransactionPreSize*7)

	txs, size = pool.getProcessableObjects(types.TransactionPreSize * 10)
	assert.Equal(t, len(txs), 0)
	assert.Equal(t, size, 0)
}

func (chain mockBlockchain) addAccount(addr common.Address, balance, nonce uint64) {
	chain.statedb.CreateAccount(addr)
	chain.statedb.SetBalance(addr, new(big.Int).SetUint64(balance))
	chain.statedb.SetNonce(addr, nonce)
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
