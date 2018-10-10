/**
* @file
* @copyright defined in go-seele/LICENSE
 */
package svm

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/core/vm"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func Test_Process_EVM(t *testing.T) {
	ctx, err := newTestContext(big.NewInt(0))
	assert.Equal(t, err, nil)

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

func Test_Process_SysContract(t *testing.T) {
	// CreateDomainName
	ctx, _ := newTestContext(big.NewInt(0))
	testBytes := []byte("seele.fan")
	ctx.Tx.Data.Payload = append([]byte{system.CmdCreateDomainName}, testBytes...) // 0x007365656c652e66616e
	ctx.Tx.Data.To = system.DomainNameContractAddress                              // 0x0000000000000000000000000000000000000101

	receipt, err := Process(ctx)
	assert.Equal(t, nil, err)
	assert.Equal(t, false, receipt.Failed)
	assert.Equal(t, ctx.Tx.Hash, receipt.TxHash)

	gasCreateDomainName := uint64(50000) // gas used to create a domain name
	assert.Equal(t, receipt.UsedGas, gasCreateDomainName+ctx.Tx.IntrinsicGas())
	assert.Equal(t, new(big.Int).SetUint64(receipt.TotalFee), new(big.Int).Add(usedGasFee(gasCreateDomainName), ctx.Tx.Data.Fee))

	// DomainNameOwner
	ctx1 := ctx
	ctx1.Tx.Data.Payload = append([]byte{system.CmdGetDomainNameOwner}, testBytes...) // 0x017365656c652e66616e

	receipt1, err1 := Process(ctx1)
	assert.Equal(t, nil, err1)
	assert.Equal(t, false, receipt1.Failed)
	assert.Equal(t, ctx1.Tx.Hash, receipt1.TxHash)

	gasDomainNameCreator := uint64(100000) // gas used to query the creator of given domain name
	assert.Equal(t, receipt1.UsedGas, gasDomainNameCreator+ctx.Tx.IntrinsicGas())
	assert.Equal(t, new(big.Int).SetUint64(receipt1.TotalFee), new(big.Int).Add(usedGasFee(gasDomainNameCreator), ctx1.Tx.Data.Fee))

	// Do not transfer the amount of the run error
	ctx2 := ctx1
	ctx2.Tx.Data.Payload = append([]byte{system.CmdGetDomainNameOwner + 1}, testBytes...) // 0x007365656c652e66616e
	ctx2.Tx.Data.Amount = big.NewInt(7)

	fromOriginalBalance := ctx2.Statedb.GetBalance(ctx2.Tx.Data.From)
	toOriginalBalance := ctx2.Statedb.GetBalance(ctx2.Tx.Data.To)
	receipt2, err2 := Process(ctx2)
	fromCurrentBalance := ctx2.Statedb.GetBalance(ctx2.Tx.Data.From)
	toCurrentBalance := ctx2.Statedb.GetBalance(ctx2.Tx.Data.To)
	assert.Equal(t, nil, err2)
	assert.Equal(t, true, receipt2.Failed)
	assert.Equal(t, fromCurrentBalance.Add(fromCurrentBalance, new(big.Int).SetUint64(receipt2.TotalFee)), fromOriginalBalance)
	assert.Equal(t, toOriginalBalance, toCurrentBalance)
}

func Test_Process_ErrInsufficientBalance(t *testing.T) {
	// get the tx total fee
	ctx, _ := newTestContext(big.NewInt(1))
	receipt, err := Process(ctx)
	assert.Equal(t, err, nil)
	totalFee := receipt.TotalFee

	// cannot apply the tx
	ctx1, _ := newTestContext(big.NewInt(1))
	balanceF1 := big.NewInt(0)
	ctx1.Statedb.SetBalance(ctx1.Tx.Data.From, balanceF1)
	receipt1, err1 := Process(ctx1)
	assert.Equal(t, err1, vm.ErrInsufficientBalance)
	assert.Empty(t, receipt1)

	// can apply the tx but not enough fee
	ctx2, _ := newTestContext(big.NewInt(1))
	balanceF2 := big.NewInt(0).Sub(big.NewInt(0).SetUint64(totalFee), ctx2.Tx.Data.Fee)
	ctx2.Statedb.SetBalance(ctx2.Tx.Data.From, balanceF2)
	receipt2, err2 := Process(ctx2)
	assert.Equal(t, err2, vm.ErrInsufficientBalance)
	assert.Empty(t, receipt2)

	// nonce not changed
	nonce1 := ctx1.Statedb.GetNonce(ctx1.Tx.Data.From)
	assert.Equal(t, nonce1, ctx1.Tx.Data.AccountNonce)
	nonce2 := ctx2.Statedb.GetNonce(ctx2.Tx.Data.From)
	assert.Equal(t, nonce2, ctx2.Tx.Data.AccountNonce)

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

func newTestContext(amount *big.Int) (*Context, error) {
	statedb, bcStore, address, dispose := preprocessContract(fromBalance, 38)
	defer dispose()

	coinbase := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(coinbase)
	header := newTestBlockHeader(coinbase)
	// Create contract tx, please refer to the code content in contract/solidity/simple_storage.sol
	code := mustHexToBytes("0x608060405234801561001057600080fd5b50600560008190555060df806100276000396000f3006080604052600436106049576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806360fe47b114604e5780636d4ce63c146078575b600080fd5b348015605957600080fd5b5060766004803603810190808035906020019092919050505060a0565b005b348015608357600080fd5b50608a60aa565b6040518082815260200191505060405180910390f35b8060008190555050565b600080549050905600a165627a7a723058207f6dc43a0d648e9f5a0cad5071cde46657de72eb87ab4cded53a7f1090f51e6d0029")
	tx, err := types.NewContractTransaction(address, amount, big.NewInt(1), math.MaxUint64, 38, code)

	return &Context{
		Tx:          tx,
		TxIndex:     8,
		Statedb:     statedb,
		BlockHeader: header,
		BcStore:     bcStore,
	}, err
}

func Benchmark_CreateContract_EVM(b *testing.B) {
	ctx, _ := newTestContext(big.NewInt(0))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Process(ctx)
	}
}

func Benchmark_CallContract_EVM(b *testing.B) {
	ctx, _ := newTestContext(big.NewInt(0))
	receipt, _ := Process(ctx)
	contractAddr := common.BytesToAddress(receipt.ContractAddress)

	// Call contract tx: SimpleStorage.get(), it returns 5 as initialized in constructor.
	input := mustHexToBytes("0x6d4ce63c")
	amount, fee, nonce := big.NewInt(0), big.NewInt(1), uint64(38)
	callContractTx, _ := types.NewMessageTransaction(ctx.Tx.Data.From, contractAddr, amount, fee, math.MaxUint64, nonce, input)
	ctx.Tx = callContractTx
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Process(ctx)
	}
}
