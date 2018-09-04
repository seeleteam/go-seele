/**
* @file
* @copyright defined in go-seele/LICENSE
 */
package svm

import (
	"math/big"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/core/vm"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func Test_Process(t *testing.T) {
	ctx := newTestContext(t, big.NewInt(0))

	receipt, err := Process(ctx)
	assert.Equal(t, err, nil)
	assert.Equal(t, receipt.Failed, false)
	assert.Equal(t, receipt.TxHash, ctx.Tx.CalculateHash())
	assert.Equal(t, receipt.ContractAddress, crypto.CreateAddress(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce).Bytes())

	// nonce + 1
	nonce := ctx.Statedb.GetNonce(ctx.Tx.Data.From)
	assert.Equal(t, nonce, ctx.Tx.Data.AccountNonce+1)

	// add fee to coinbase and sub fee from tx.from
	balanceC := ctx.Statedb.GetBalance(ctx.BlockHeader.Creator)
	assert.Equal(t, big.NewInt(0).SetUint64(receipt.TotalFee), balanceC)

	balanceF := ctx.Statedb.GetBalance(ctx.Tx.Data.From)
	assert.Equal(t, big.NewInt(0).Sub(big.NewInt(0).SetUint64(fromBalance-receipt.TotalFee), ctx.Tx.Data.Amount), balanceF)

	// postState
	postState, err1 := ctx.Statedb.Hash()
	assert.Equal(t, err1, nil)
	assert.Equal(t, postState, receipt.PostState)

	// logs
	logs := make([]*types.Log, 0)
	assert.Equal(t, logs, receipt.Logs)
}

func Test_Process_ErrInsufficientBalance(t *testing.T) {
	// get the tx total fee
	ctx := newTestContext(t, big.NewInt(1))
	receipt, err := Process(ctx)
	assert.Equal(t, err, nil)
	totalFee := receipt.TotalFee

	// cannot apply the tx
	ctx1, balanceF1 := newTestContext(t, big.NewInt(1)), big.NewInt(0)
	ctx1.Statedb.SetBalance(ctx1.Tx.Data.From, balanceF1)
	receipt1, err1 := Process(ctx1)
	assert.Equal(t, err1, vm.ErrInsufficientBalance)
	assert.Empty(t, receipt1)

	// can apply the tx but not enough fee
	ctx2 := newTestContext(t, big.NewInt(1))
	balanceF2 := big.NewInt(0).Sub(big.NewInt(0).SetUint64(totalFee), ctx2.Tx.Data.Fee)
	ctx2.Statedb.SetBalance(ctx2.Tx.Data.From, balanceF2)
	receipt2, err2 := Process(ctx2)
	assert.Equal(t, err2, vm.ErrInsufficientBalance)
	assert.Empty(t, receipt2)

	// nonce not changed
	nonce1 := ctx1.Statedb.GetNonce(ctx1.Tx.Data.From)
	assert.Equal(t, nonce1, ctx1.Tx.Data.AccountNonce)
	nonce2 := ctx2.Statedb.GetNonce(ctx2.Tx.Data.From)
	assert.Equal(t, nonce2, ctx2.Tx.Data.AccountNonce+1)

	// balance not changed
	balanceC1 := ctx1.Statedb.GetBalance(ctx1.BlockHeader.Creator)
	assert.Equal(t, balanceC1, big.NewInt(0))
	balanceF1Now := ctx1.Statedb.GetBalance(ctx1.Tx.Data.From)
	assert.Equal(t, balanceF1Now, balanceF1)
	balanceC2 := ctx2.Statedb.GetBalance(ctx2.BlockHeader.Creator)
	assert.Equal(t, balanceC2, big.NewInt(0))
	balanceF2Now := ctx2.Statedb.GetBalance(ctx2.Tx.Data.From)
	assert.Equal(t, balanceF2Now, balanceF2)
}

func mustHexToBytes(hex string) []byte {
	code, err := hexutil.HexToBytes(hex)
	if err != nil {
		panic(err)
	}

	return code
}

// preprocessContract creates the contract tx dependent state DB, blockchain store
// and a default account with specified balance and nonce.
func preprocessContract(balance, nonce uint64) (*state.Statedb, store.BlockchainStore, common.Address, func()) {
	db, dispose := leveldb.NewTestDatabase()

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		dispose()
		panic(err)
	}

	// Create a default account to test contract.
	addr := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(addr)
	statedb.SetBalance(addr, new(big.Int).SetUint64(balance))
	statedb.SetNonce(addr, nonce)

	return statedb, store.NewBlockchainDatabase(db), addr, func() {
		dispose()
	}
}

func newTestBlockHeader(coinbase common.Address) *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: crypto.MustHash("block previous hash"),
		Creator:           coinbase,
		StateHash:         crypto.MustHash("state root hash"),
		TxHash:            crypto.MustHash("tx root hash"),
		ReceiptHash:       crypto.MustHash("receipt root hash"),
		Difficulty:        big.NewInt(38),
		Height:            666,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Nonce:             10,
		ExtraData:         make([]byte, 0),
	}
}

var fromBalance = uint64(1000 * common.SeeleToFan.Uint64())

func newTestContext(t *testing.T, amount *big.Int) *Context {
	statedb, bcStore, address, dispose := preprocessContract(fromBalance, 38)
	defer dispose()

	coinbase := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(coinbase)
	header := newTestBlockHeader(coinbase)
	// Create contract tx, please refer to the code content in contract/solidity/simple_storage.sol
	code := mustHexToBytes("0x608060405234801561001057600080fd5b50600560008190555060df806100276000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058207f6dc43a0d648e9f5a0cad5071cde46657de72eb87ab4cded53a7f1090f51e6d0029")
	tx, err := types.NewContractTransaction(address, amount, big.NewInt(1), 38, code)
	assert.Equal(t, err, nil)

	return &Context{
		Tx:          tx,
		TxIndex:     8,
		Statedb:     statedb,
		BlockHeader: header,
		BcStore:     bcStore,
	}
}
