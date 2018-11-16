/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func Test_verifyTx(t *testing.T) {
	req, reqBytes := newTestRelayRequest(t)
	ctx := newTestRelayContext(t)

	// simulate svm transfer amount to contract address
	ctx.statedb.AddBalance(BTCRelayContractAddress, ctx.tx.Data.Amount)
	blocks := getBTCBlocks(ctx.statedb)
	oriRelayerBalance := ctx.statedb.GetBalance(blocks[req.BlockHeaderHex].Relayer)
	oriContractBalance := ctx.statedb.GetBalance(BTCRelayContractAddress)

	ok1, err1 := verifyTx(reqBytes, ctx)
	assert.NoError(t, err1)
	assert.Equal(t, success, ok1)

	curRelayerBalance := ctx.statedb.GetBalance(blocks[req.BlockHeaderHex].Relayer)
	curContractBalance := ctx.statedb.GetBalance(BTCRelayContractAddress)
	assert.Equal(t, curContractBalance.Add(curContractBalance, ctx.tx.Data.Amount), oriContractBalance)
	assert.Equal(t, curRelayerBalance.Sub(curRelayerBalance, ctx.tx.Data.Amount), oriRelayerBalance)

	// invalid request
	ok2, err2 := verifyTx([]byte{0, 1, 2, 3}, ctx)
	assert.Error(t, err2)
	assert.Equal(t, failure, ok2)

	// confirmation is less than 6
	for _, block := range blocks {
		if block.Height < preBlockHeight-6 {
			req1, _ := newTestRelayRequest(t)
			req1.BlockHeaderHex = block.BlockHeaderHex
			reqBytes1, err1 := json.Marshal(req1)
			assert.Equal(t, nil, err1)

			ok2, err2 := verifyTx(reqBytes1, ctx)
			assert.Error(t, err2)
			assert.Equal(t, failure, ok2)
			break
		}
	}

	// fee is not enough
	ctx.tx.Data.Amount = big.NewInt(0)
	ok4, err4 := verifyTx(reqBytes, ctx)
	assert.Error(t, err4)
	assert.Equal(t, failure, ok4)
	ctx.tx.Data.Amount = big.NewInt(1)

	// tx doesn't exist
	req.TxHex = "asdf"
	reqBytes2, err21 := json.Marshal(req)
	assert.NoError(t, err21)
	ok6, err6 := verifyTx(reqBytes2, ctx)
	assert.NoError(t, err6)
	assert.Equal(t, failure, ok6)

	// blockheader doesn't exist
	req.BlockHeaderHex = "asdf"
	reqBytes1, err11 := json.Marshal(req)
	assert.NoError(t, err11)
	ok5, err5 := verifyTx(reqBytes1, ctx)
	assert.NoError(t, err5)
	assert.Equal(t, failure, ok5)
}

func Test_relayTx(t *testing.T) {
	req, reqBytes := newTestRelayRequest(t)
	ctx := newTestRelayContext(t)

	// simulate svm transfer amount to contract address
	ctx.statedb.AddBalance(BTCRelayContractAddress, ctx.tx.Data.Amount)
	blocks := getBTCBlocks(ctx.statedb)
	oriRelayerBalance := ctx.statedb.GetBalance(blocks[req.BlockHeaderHex].Relayer)
	oriContractBalance := ctx.statedb.GetBalance(BTCRelayContractAddress)

	ok1, err1 := relayTx(reqBytes, ctx)
	assert.NoError(t, err1)
	assert.Equal(t, success, ok1)

	curRelayerBalance := ctx.statedb.GetBalance(blocks[req.BlockHeaderHex].Relayer)
	curContractBalance := ctx.statedb.GetBalance(BTCRelayContractAddress)
	assert.Equal(t, curContractBalance.Add(curContractBalance, ctx.tx.Data.Amount), oriContractBalance)
	assert.Equal(t, curRelayerBalance.Sub(curRelayerBalance, ctx.tx.Data.Amount), oriRelayerBalance)

	// verifyTx error
	ok2, err2 := relayTx([]byte{0, 1, 2, 3}, ctx)
	assert.Error(t, err2)
	assert.Equal(t, failure, ok2)
}

func Test_storeBlockHeader(t *testing.T) {
	req, _ := newTestRelayRequest(t)
	req.PreviousBlockHex = req.BlockHeaderHex
	req.BlockHeaderHex = crypto.MustGenerateRandomAddress().Hex()
	req.TxHexs = []string{crypto.MustGenerateRandomAddress().Hex()}
	req.Height++
	reqBytes, err := json.Marshal(req)
	assert.NoError(t, err)

	ctx := newTestRelayContext(t)
	ok1, err1 := storeBlockHeader(reqBytes, ctx)
	assert.NoError(t, err1)
	assert.Equal(t, success, ok1)
	assert.Equal(t, req.Height, preBlockHeight)

	blocks := getBTCBlocks(ctx.statedb)
	block := blocks[req.BlockHeaderHex]
	assert.Equal(t, req.BlockHeaderHex, block.BlockHeaderHex)
	assert.Equal(t, req.Height, block.Height)
	assert.Equal(t, req.PreviousBlockHex, block.PreviousBlockHex)
	assert.Equal(t, req.TxHexs, block.TxHexs)

	// invalid request
	ok2, err2 := storeBlockHeader([]byte{0, 1, 2, 3}, ctx)
	assert.Error(t, err2)
	assert.Equal(t, failure, ok2)

	// repeat store
	ok3, err3 := storeBlockHeader(reqBytes, ctx)
	assert.Error(t, err3)
	assert.Equal(t, failure, ok3)
}

func Test_getBlockHeader(t *testing.T) {
	req, _ := newTestRelayRequest(t)
	headerBytes, err := hexutil.HexToBytes(req.BlockHeaderHex)
	assert.NoError(t, err)
	// simulate svm transfer amount to contract address
	ctx := newTestRelayContext(t)
	ctx.statedb.AddBalance(BTCRelayContractAddress, ctx.tx.Data.Amount)
	blocks := getBTCBlocks(ctx.statedb)
	oriRelayerBalance := ctx.statedb.GetBalance(blocks[req.BlockHeaderHex].Relayer)
	oriContractBalance := ctx.statedb.GetBalance(BTCRelayContractAddress)

	ok, err1 := getBlockHeader(headerBytes, ctx)
	assert.NoError(t, err1)
	assert.Equal(t, success, ok)

	curRelayerBalance := ctx.statedb.GetBalance(blocks[req.BlockHeaderHex].Relayer)
	curContractBalance := ctx.statedb.GetBalance(BTCRelayContractAddress)
	assert.Equal(t, curContractBalance.Add(curContractBalance, ctx.tx.Data.Amount), oriContractBalance)
	assert.Equal(t, curRelayerBalance.Sub(curRelayerBalance, ctx.tx.Data.Amount), oriRelayerBalance)

	// fee is not enough
	ctx.tx.Data.Amount = big.NewInt(0)
	ok2, err2 := getBlockHeader(headerBytes, ctx)
	assert.Error(t, err2)
	assert.Equal(t, failure, ok2)
	ctx.tx.Data.Amount = big.NewInt(1)

	// blockheader doesn't exist
	ok3, err3 := getBlockHeader([]byte("asdf"), ctx)
	assert.NoError(t, err3)
	assert.Equal(t, failure, ok3)
}

func newTestRelayContext(t *testing.T) *Context {
	dbPath := filepath.Join(common.GetTempFolder(), ".newTestRelayContext")
	db, err := leveldb.NewLevelDB(dbPath)
	if err != nil {
		panic(err)
	}

	defer func() {
		db.Close()
		os.RemoveAll(dbPath)
	}()

	ctx := newTestContext(db, BTCRelayContractAddress)
	req, reqBytes := newTestRelayRequest(t)
	storeBlockHeader(reqBytes, ctx)
	for height := uint64(0); height < 10; height++ {
		req.PreviousBlockHex = req.BlockHeaderHex
		req.BlockHeaderHex = crypto.MustGenerateRandomAddress().Hex()
		req.TxHexs = []string{crypto.MustGenerateRandomAddress().Hex()}
		req.Height = height
		reqBytes, _ := json.Marshal(req)
		storeBlockHeader(reqBytes, ctx)
	}

	return ctx
}

func newTestRelayRequest(t *testing.T) (*RelayRequest, []byte) {
	req := &RelayRequest{
		BTCBlock: BTCBlock{
			BlockHeaderHex:   "0x0f9188f13cb7b2c71f2a335e3a4fc328bf5beb436012afca590b1a11466e2206",
			Height:           10,
			PreviousBlockHex: "0x2a53daccb2587168ee58e385e7ba274de1ae37c5d21b6b709a81d019fa2a65b4",
			TxHexs:           []string{"0x4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b"},
		},
		TxHex:        "0x4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b",
		RelayAddress: common.EmptyAddress,
	}
	reqBytes, err := json.Marshal(req)
	assert.NoError(t, err)
	assert.NotEmpty(t, req.String())

	return req, reqBytes
}
