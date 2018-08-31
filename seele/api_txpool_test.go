package seele

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

func Test_GetTxByHash(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GetTxByHash")
	if common.FileOrFolderExists(dbPath) {
		os.RemoveAll(dbPath)
	}
	api := newTestAPI(t, dbPath)

	// add tx
	tx1 := newTestTx(t, api.s, 1, 2, 1)
	err := api.s.txPool.AddTransaction(tx1)
	assert.Equal(t, err, nil)

	// verify pool tx
	poolAPI := NewPrivateTransactionPoolAPI(api.s)
	outputs, err := poolAPI.GetTransactionByHash(tx1.Hash.ToHex())
	assert.Equal(t, err, nil)
	assert.Equal(t, tx1.Hash.ToHex(), outputs["transaction"].(map[string]interface{})["hash"].(string))
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
	assert.Equal(t, outputs["transaction"].(map[string]interface{})["hash"].(string), tx1.Hash.ToHex())
	assert.Equal(t, outputs["status"], "block")
	assert.Equal(t, outputs["blockHash"], block.HeaderHash.ToHex())
	assert.Equal(t, outputs["blockHeight"], block.Header.Height)
	assert.Equal(t, outputs["txIndex"], uint(0))
}

func Test_GetReceiptByHash(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GetReceiptByHash")
	if common.FileOrFolderExists(dbPath) {
		os.RemoveAll(dbPath)
	}
	api := newTestTxPoolAPI(t, dbPath)

	// save receipts to block
	tx1 := newTestTx(t, api.s, 1, 2, 1)
	receipts := []*types.Receipt{
		&types.Receipt{
			Result:    []byte("result"),
			PostState: common.StringToHash("post state"),
			Logs:      []*types.Log{&types.Log{}, &types.Log{}, &types.Log{}},
			TxHash:    tx1.Hash,
			UsedGas:   123,
			TotalFee:  456,
		},
		&types.Receipt{
			Result:    []byte("result"),
			PostState: common.StringToHash("post state"),
			Logs:      []*types.Log{&types.Log{}, &types.Log{}, &types.Log{}},
			TxHash:    tx1.Hash,
			UsedGas:   789,
			TotalFee:  120,
		},
	}
	block := api.s.chain.CurrentBlock()
	block.Header.Height++
	block.Header.PreviousBlockHash = block.HeaderHash
	block.Transactions = []*types.Transaction{tx1}
	block.HeaderHash = block.Header.Hash()
	err := api.s.chain.GetStore().PutBlock(block, block.Header.Difficulty, true)
	assert.Equal(t, err, nil)
	err = api.s.chain.GetStore().PutReceipts(block.HeaderHash, receipts)
	assert.Equal(t, err, nil)

	// verify block receipt
	poolAPI := NewPrivateTransactionPoolAPI(api.s)
	outputs, err := poolAPI.GetReceiptByTxHash(tx1.Hash.ToHex())
	assert.Equal(t, err, nil)
	assert.Equal(t, outputs["result"], hexutil.BytesToHex(receipts[0].Result))
	assert.Equal(t, outputs["failed"], false)
	assert.Equal(t, outputs["poststate"], receipts[0].PostState.ToHex())
	assert.Equal(t, outputs["txhash"], tx1.Hash.ToHex())
	assert.Equal(t, outputs["usedGas"], receipts[0].UsedGas)
	assert.Equal(t, outputs["totalFee"], receipts[0].TotalFee)
}

func newTestTx(t *testing.T, s *SeeleService, amount, fee int64, nonce uint64) *types.Transaction {
	statedb, err := s.chain.GetCurrentState()
	assert.Equal(t, err, nil)

	// set initial balance
	fromAddress, fromPrivKey, err := crypto.GenerateKeyPair()
	assert.Equal(t, err, nil)
	statedb.CreateAccount(*fromAddress)
	statedb.SetBalance(*fromAddress, common.SeeleToFan)
	statedb.SetNonce(*fromAddress, nonce-1)

	err = storeStatedb(t, s, statedb)
	assert.Equal(t, err, nil)

	toAddress := crypto.MustGenerateShardAddress(fromAddress.Shard())

	tx, err := types.NewTransaction(*fromAddress, *toAddress, big.NewInt(amount), big.NewInt(fee), nonce)
	assert.Equal(t, err, nil)

	tx.Sign(fromPrivKey)
	return tx
}

func storeStatedb(t *testing.T, s *SeeleService, statedb *state.Statedb) error {
	batch := s.accountStateDB.NewBatch()
	block := s.chain.CurrentBlock()
	block.Header.StateHash, _ = statedb.Commit(batch)
	block.Header.Height++
	block.Header.PreviousBlockHash = block.HeaderHash
	block.HeaderHash = block.Header.Hash()
	s.chain.GetStore().PutBlock(block, big.NewInt(1), true)
	return batch.Commit()
}

func newTestTxPoolAPI(t *testing.T, dbPath string) *PrivateTransactionPoolAPI {
	conf := getTmpConfig()
	serviceContext := ServiceContext{
		DataDir: dbPath,
	}

	var key interface{} = "ServiceContext"
	ctx := context.WithValue(context.Background(), key, serviceContext)
	dataDir := ctx.Value("ServiceContext").(ServiceContext).DataDir
	defer os.RemoveAll(dataDir)

	log := log.GetLogger("seele")
	ss, err := NewSeeleService(ctx, conf, log)
	assert.Equal(t, err, nil)

	return NewPrivateTransactionPoolAPI(ss)
}
