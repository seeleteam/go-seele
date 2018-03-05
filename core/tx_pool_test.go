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
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

func randomAddress(t *testing.T) common.Address {
	address, err := common.GenerateRandomAddress()
	if err != nil {
		t.Fatalf("Failed to generate random address, error = %s", err.Error())
	}

	return *address
}

func randomKey(t *testing.T) *ecdsa.PrivateKey {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate ECDSA private key, error = %s", err.Error())
	}

	return privKey
}

func newTestTx(t *testing.T, amount int64, nonce uint64) *types.Transaction {
	tx := types.NewTransaction(randomAddress(t), randomAddress(t), big.NewInt(amount), nonce)
	privKey := randomKey(t)
	tx.Sign(privKey)
	return tx
}

func Test_TransactionPool_Add_ValidTx(t *testing.T) {
	pool := NewTransactionPool(*DefaultTxPoolConfig())
	tx := newTestTx(t, 10, 100)

	added, err := pool.AddTransaction(tx)

	assert.Equal(t, added, true)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, len(pool.hashToTxMap), 1)
}

func Test_TransactionPool_Add_InvalidTx(t *testing.T) {
	pool := NewTransactionPool(*DefaultTxPoolConfig())
	tx := newTestTx(t, 10, 100)

	// Change the amount in tx.
	tx.Data.Amount.SetInt64(20)
	added, err := pool.AddTransaction(tx)

	assert.Equal(t, added, false)
	if err == nil {
		t.Fatal("The error is nil when add invalid tx to pool.")
	}
}

func Test_TransactionPool_Add_DuplicateTx(t *testing.T) {
	pool := NewTransactionPool(*DefaultTxPoolConfig())
	tx := newTestTx(t, 10, 100)

	added, err := pool.AddTransaction(tx)
	assert.Equal(t, added, true)
	assert.Equal(t, err, error(nil))

	added, err = pool.AddTransaction(tx)
	assert.Equal(t, added, false)
	assert.Equal(t, err, errTxHashExists)
}

func Test_TransactionPool_Add_PoolFull(t *testing.T) {
	config := DefaultTxPoolConfig()
	config.Capacity = 1
	pool := NewTransactionPool(*config)

	tx1 := newTestTx(t, 10, 100)
	tx2 := newTestTx(t, 20, 101)

	added, err := pool.AddTransaction(tx1)
	assert.Equal(t, added, true)
	assert.Equal(t, err, error(nil))

	added, err = pool.AddTransaction(tx2)
	assert.Equal(t, added, false)
	assert.Equal(t, err, errTxPoolFull)
}
