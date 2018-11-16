/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	api2 "github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/consensus/factory"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
)

func newTestSeeleBackend() *SeeleBackend {
	seeleService := newTestSeeleService()
	return &SeeleBackend{seeleService}
}

func Test_SeeleBackend_GetBlock(t *testing.T) {
	seeleBackend := newTestSeeleBackend()
	defer seeleBackend.s.Stop()

	block, err := seeleBackend.GetBlock(common.EmptyHash, -1)
	assert.Equal(t, err, nil)
	assert.Equal(t, block.Header.Height, uint64(0))

	block1, err := seeleBackend.GetBlock(block.HeaderHash, -1)
	assert.Equal(t, err, nil)
	assert.Equal(t, block1.HeaderHash, block.HeaderHash)

	block2, err := seeleBackend.GetBlock(common.EmptyHash, 0)
	assert.Equal(t, err, nil)
	assert.Equal(t, block2.Header.Height, uint64(0))
	assert.Equal(t, block2.HeaderHash, block.HeaderHash)
}

func Test_GetReceiptByHash(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GetReceiptByHash")
	api := newTestTxPoolAPI(t, dbPath)
	defer func() {
		api.s.Stop()
		os.RemoveAll(dbPath)
	}()

	// save receipts to block
	tx1 := newTestTx(t, api.s, 1, 2, 1)
	receipts := newTestTxReceipt(tx1)

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
	poolAPI := NewSeeleBackend(api.s)
	receipt, err := poolAPI.GetReceiptByTxHash(tx1.Hash)
	assert.Equal(t, err, nil)
	outputs, err := api2.PrintableReceipt(receipt)
	assert.Equal(t, err, nil)
	assert.Equal(t, outputs["result"], hexutil.BytesToHex(receipts[0].Result))
	assert.Equal(t, outputs["failed"], false)
	assert.Equal(t, outputs["poststate"], receipts[0].PostState.Hex())
	assert.Equal(t, outputs["txhash"], tx1.Hash.Hex())
	assert.Equal(t, outputs["usedGas"], receipts[0].UsedGas)
	assert.Equal(t, outputs["totalFee"], receipts[0].TotalFee)
}

func newTestTx(t *testing.T, s *SeeleService, amount, price int64, nonce uint64) *types.Transaction {
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
	tx, err := types.NewTransaction(*fromAddress, *toAddress, big.NewInt(amount), big.NewInt(price), nonce)
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

func newTestTxPoolAPI(t *testing.T, dbPath string) *TransactionPoolAPI {
	conf := getTmpConfig()
	serviceContext := ServiceContext{
		DataDir: dbPath,
	}

	var key interface{} = "ServiceContext"
	ctx := context.WithValue(context.Background(), key, serviceContext)
	log := log.GetLogger("seele")
	ss, err := NewSeeleService(ctx, conf, log, factory.MustGetConsensusEngine(common.Sha256Algorithm), nil)
	if err != nil {
		panic("new seele service error")
	}
	return NewTransactionPoolAPI(ss)
}

func newTestTxReceipt(tx1 *types.Transaction) []*types.Receipt {
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
	return receipts
}
