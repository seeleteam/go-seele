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
	fromPrivKey, fromAddress := randomAccount(t)
	_, toAddress := randomAccount(t)

	tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(1), nonce)
	tx.Sign(fromPrivKey)

	return &pooledTx{tx, PENDING, time.Now().Unix()}
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
		config:          *config,
		chain:           chain,
		hashToTxMap:     make(map[common.Hash]*pooledTx),
		accountToTxsMap: make(map[common.Address]*txCollection),
		lastHeader:      common.EmptyHash,
		log:             log.GetLogger("test", true),
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

func Test_TransactionPool_GetProcessableTransactions(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

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
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.Transaction)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, len(pool.accountToTxsMap), 1)

	pool.removeTransaction(poolTx.Hash)
	assert.Equal(t, len(pool.accountToTxsMap), 0)
}

func Test_TransactionPool_UpdateTransactionStatus(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	poolTx := newTestPoolTx(t, 10, 100)
	chain.addAccount(poolTx.Data.From, 20, 100)

	err := pool.AddTransaction(poolTx.Transaction)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, len(pool.accountToTxsMap), 1)

	pool.UpdateTransactionStatus(poolTx.Hash, PROCESSING)
	assert.Equal(t, pool.hashToTxMap[poolTx.Hash].txStatus, PROCESSING)
	assert.Equal(t, pool.accountToTxsMap[poolTx.Data.From].nonceToTxMap[100].txStatus, PROCESSING)
}

func Test_GetRejectTransacton(t *testing.T) {
	log := log.GetLogger("test", true)
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bc := newTestBlockchain(db)
	b1 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 3, 0)
	bc.WriteBlock(b1)

	b2 := newTestBlock(bc, bc.genesisBlock.HeaderHash, 1, 2, 0)
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
	assert.Equal(t, len(pool.accountToTxsMap), 1)

	for _, ptx := range pool.hashToTxMap {
		ptx.timestamp = ptx.timestamp - 10
	}

	pool.RemoveTransactions()
	assert.Equal(t, len(pool.hashToTxMap), 1)
	assert.Equal(t, len(pool.accountToTxsMap), 1)

	for _, ptx := range pool.hashToTxMap {
		ptx.timestamp = ptx.timestamp - overTimeInterval
	}

	pool.RemoveTransactions()
	assert.Equal(t, len(pool.hashToTxMap), 0)
	assert.Equal(t, len(pool.accountToTxsMap), 0)
}
