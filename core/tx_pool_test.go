/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
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
	chain.addAccount(poolTx.Account(), 20, 100)

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
	chain.addAccount(poolTx.Account(), 500000, 100)

	err := pool.AddObject(poolTx.poolObject)
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
