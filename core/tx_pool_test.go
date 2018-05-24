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
)

func randomAccount(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	privKey, keyErr := crypto.GenerateKey()
	if keyErr != nil {
		t.Fatalf("Failed to generate ECDSA private key, error = %s", keyErr.Error())
	}

	hexAddress := crypto.PubkeyToString(&privKey.PublicKey)

	return privKey, common.HexMustToAddres(hexAddress)
}

func newTestPoolTx(t *testing.T, amount int64, nonce uint64) *poolTransaction {
	fromPrivKey, fromAddress := randomAccount(t)
	_, toAddress := randomAccount(t)

	tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(0), nonce)
	tx.Sign(fromPrivKey)

	return &poolTransaction{
		transaction: tx,
		txStatus:    PENDING,
	}
}

type mockBlockchain struct {
	statedb *state.Statedb
}

func newMockBlockchain() *mockBlockchain {
	statedb, err := state.NewStatedb(common.EmptyHash, nil)
	if err != nil {
		panic(err)
	}

	return &mockBlockchain{statedb}
}

func (chain mockBlockchain) CurrentState() *state.Statedb {
	return chain.statedb
}

func (chain mockBlockchain) GetStore() store.BlockchainStore {
	return chain.GetStore()
}

func (chain mockBlockchain) addAccount(addr common.Address, balance, nonce uint64) {
	stateObj := chain.statedb.GetOrNewStateObject(addr)
	stateObj.SetAmount(new(big.Int).SetUint64(balance))
	stateObj.SetNonce(nonce)
}

func Test_TransactionPool_Add_ValidTx(t *testing.T) {
	chain := newMockBlockchain()
	pool := NewTransactionPool(*DefaultTxPoolConfig(), chain)
	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.transaction.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.transaction)

	assert.Equal(t, err, error(nil))
	assert.Equal(t, len(pool.hashToTxMap), 1)
}

func Test_TransactionPool_Add_InvalidTx(t *testing.T) {
	chain := newMockBlockchain()
	pool := NewTransactionPool(*DefaultTxPoolConfig(), chain)
	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.transaction.Data.From, 20, 100)

	// Change the amount in tx.
	poolTx.transaction.Data.Amount.SetInt64(20)
	err := pool.AddTransaction(poolTx.transaction)

	if err == nil {
		t.Fatal("The error is nil when add invalid tx to pool.")
	}
}

func Test_TransactionPool_Add_DuplicateTx(t *testing.T) {
	chain := newMockBlockchain()
	pool := NewTransactionPool(*DefaultTxPoolConfig(), chain)
	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.transaction.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.transaction)
	assert.Equal(t, err, error(nil))

	err = pool.AddTransaction(poolTx.transaction)
	assert.Equal(t, err, errTxHashExists)
}

func Test_TransactionPool_Add_PoolFull(t *testing.T) {
	config := DefaultTxPoolConfig()
	config.Capacity = 1
	chain := newMockBlockchain()
	pool := NewTransactionPool(*config, chain)

	poolTx1 := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx1.transaction.Data.From, 20, 100)
	poolTx2 := newTestPoolTx(t, 20, 101)
	chain.addAccount(poolTx2.transaction.Data.From, 20, 101)

	err := pool.AddTransaction(poolTx1.transaction)
	assert.Equal(t, err, error(nil))

	err = pool.AddTransaction(poolTx2.transaction)
	assert.Equal(t, err, errTxPoolFull)
}

func Test_TransactionPool_GetTransaction(t *testing.T) {
	chain := newMockBlockchain()
	pool := NewTransactionPool(*DefaultTxPoolConfig(), chain)
	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.transaction.Data.From, 20, 100)

	pool.AddTransaction(poolTx.transaction)

	assert.Equal(t, pool.GetTransaction(poolTx.transaction.Hash), poolTx.transaction)
}

func newTestAccountTxs(t *testing.T, amounts []int64, nonces []uint64) (common.Address, []*types.Transaction) {
	if len(amounts) != len(nonces) || len(amounts) == 0 {
		t.Fatal()
	}

	fromPrivKey, fromAddress := randomAccount(t)
	txs := make([]*types.Transaction, 0, len(amounts))

	for i, amount := range amounts {
		_, toAddress := randomAccount(t)

		tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(0), nonces[i])
		tx.Sign(fromPrivKey)

		txs = append(txs, tx)
	}

	return fromAddress, txs
}

func Test_TransactionPool_GetProcessableTransactions(t *testing.T) {
	chain := newMockBlockchain()
	pool := NewTransactionPool(*DefaultTxPoolConfig(), chain)
	account1, txs1 := newTestAccountTxs(t, []int64{1, 2, 3}, []uint64{9, 5, 7})
	chain.addAccount(account1, 10, 5)
	account2, txs2 := newTestAccountTxs(t, []int64{1, 2, 3}, []uint64{7, 9, 5})
	chain.addAccount(account2, 10, 5)

	for _, tx := range append(txs1, txs2...) {
		pool.AddTransaction(tx)
	}

	processableTxs := pool.GetProcessableTransactions()
	assert.Equal(t, len(processableTxs), 2)

	assert.Equal(t, len(processableTxs[account1]), 3)
	assert.Equal(t, processableTxs[account1][0], txs1[1])
	assert.Equal(t, processableTxs[account1][1], txs1[2])
	assert.Equal(t, processableTxs[account1][2], txs1[0])

	assert.Equal(t, len(processableTxs[account2]), 3)
	assert.Equal(t, processableTxs[account2][0], txs2[2])
	assert.Equal(t, processableTxs[account2][1], txs2[0])
	assert.Equal(t, processableTxs[account2][2], txs2[1])
}

func Test_TransactionPool_Remove(t *testing.T) {
	config := DefaultTxPoolConfig()
	chain := newMockBlockchain()
	pool := NewTransactionPool(*config, chain)

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.transaction.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.transaction)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, len(pool.accountToTxsMap), 1)

	pool.removeTransaction(poolTx.transaction.Hash)
	assert.Equal(t, len(pool.accountToTxsMap), 0)
}

func Test_TransactionPool_ReflushTransactionStatus(t *testing.T) {
	config := DefaultTxPoolConfig()
	chain := newMockBlockchain()
	pool := NewTransactionPool(*config, chain)

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.transaction.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.transaction)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, len(pool.accountToTxsMap), 1)

	pool.ReflushTransactionStatus(poolTx.transaction.Hash, PROCESSING)
	assert.Equal(t, pool.hashToTxMap[poolTx.transaction.Hash].txStatus, PROCESSING)
	assert.Equal(t, pool.accountToTxsMap[poolTx.transaction.Data.From].nonceToTxMap[100].txStatus, PROCESSING)
}
