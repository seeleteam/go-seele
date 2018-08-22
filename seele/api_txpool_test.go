package seele

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

func Test_GetTxByHash(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GetTxByHash")
	if common.FileOrFolderExists(dbPath) {
		os.RemoveAll(dbPath)
	}
	api := newTestAPI(t, dbPath)

	// add tx
	tx1 := newTestTx(t, api, 1, 2, 1)
	err := api.s.txPool.AddTransaction(tx1)
	assert.Equal(t, err, nil)

	// verify pool tx
	poolAPI := NewPrivateTransactionPoolAPI(api.s)
	outputs, err := poolAPI.GetTransactionByHash(tx1.Hash.ToHex())
	assert.Equal(t, err, nil)
	assert.Equal(t, tx1.Hash.ToHex(), outputs["hash"])
	assert.Equal(t, outputs["status"], "pool")

	// save tx to block
	block := api.s.chain.CurrentBlock()
	block.Header.Height++
	block.Header.PreviousBlockHash = block.HeaderHash
	block.Transactions = []*types.Transaction{tx1}
	block.HeaderHash = block.Header.Hash()
	err = api.s.chain.GetStore().PutBlock(block, block.Header.Difficulty, true)
	assert.Equal(t, err, nil)

	// verify block tx
	poolAPI.s.txPool.RemoveTransaction(tx1.Hash)
	outputs, err = poolAPI.GetTransactionByHash(tx1.Hash.ToHex())
	assert.Equal(t, err, nil)
	assert.Equal(t, outputs["hash"], tx1.Hash.ToHex())
	assert.Equal(t, outputs["status"], "block")
	assert.Equal(t, outputs["blockHash"], block.HeaderHash.ToHex())
	assert.Equal(t, outputs["blockHeight"], block.Header.Height)
	assert.Equal(t, outputs["txIndex"], uint(0))
}

func newTestTx(t *testing.T, api *PublicSeeleAPI, amount, fee int64, nonce uint64) *types.Transaction {
	statedb, err := api.s.chain.GetCurrentState()
	assert.Equal(t, err, nil)

	// set initial balance
	fromAddress, fromPrivKey, err := crypto.GenerateKeyPair()
	assert.Equal(t, err, nil)
	statedb.CreateAccount(*fromAddress)
	statedb.SetBalance(*fromAddress, common.SeeleToFan)
	statedb.SetNonce(*fromAddress, nonce-1)

	err = storeStatedb(t, api, statedb)
	assert.Equal(t, err, nil)

	toAddress := crypto.MustGenerateShardAddress(fromAddress.Shard())

	tx, err := types.NewTransaction(*fromAddress, *toAddress, big.NewInt(amount), big.NewInt(fee), nonce)
	assert.Equal(t, err, nil)

	tx.Sign(fromPrivKey)
	return tx
}

func storeStatedb(t *testing.T, api *PublicSeeleAPI, statedb *state.Statedb) error {
	batch := api.s.accountStateDB.NewBatch()
	block := api.s.chain.CurrentBlock()
	block.Header.StateHash, _ = statedb.Commit(batch)
	block.Header.Height++
	block.Header.PreviousBlockHash = block.HeaderHash
	block.HeaderHash = block.Header.Hash()
	api.s.chain.GetStore().PutBlock(block, big.NewInt(1), true)
	return batch.Commit()
}
