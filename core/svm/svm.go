/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package svm

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/svm/evm"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/core/vm"
)

// Context for other vm constructs
type Context struct {
	Tx          *types.Transaction
	TxIndex     int
	Statedb     *state.Statedb
	BlockHeader *types.BlockHeader
	BcStore     store.BlockchainStore
}

// Process the tx
func Process(ctx *Context) (*types.Receipt, error) {
	// Pay intrinsic gas all the time
	gasLimit := ctx.Tx.Data.GasLimit
	intrGas := ctx.Tx.IntrinsicGas()
	if gasLimit < intrGas {
		return nil, types.ErrIntrinsicGas
	}
	leftOverGas := gasLimit - intrGas

	// init statedb and set snapshot
	var err error
	var receipt *types.Receipt
	snapshot := ctx.Statedb.Prepare(ctx.TxIndex)

	// create or execute contract
	if contract := system.GetContractByAddress(ctx.Tx.Data.To); contract != nil { // system contract
		receipt, err = processSystemContract(ctx, contract, snapshot)
	} else if ctx.Tx.IsCrossShardTx() && !ctx.Tx.Data.To.IsEVMContract() { // cross shard tx
		return processCrossShardTransaction(ctx, snapshot)
	} else { // evm
		receipt, err = processEvmContract(ctx, leftOverGas)
	}

	// Gas is not enough
	if err == vm.ErrInsufficientBalance {
		return nil, err
	}

	receipt.UsedGas += intrGas

	if err != nil {
		ctx.Statedb.RevertToSnapshot(snapshot)
		receipt.Failed = true
		receipt.Result = []byte(err.Error())
	}

	return handleFee(ctx, receipt, snapshot)
}

func processCrossShardTransaction(ctx *Context, snapshot int) (*types.Receipt, error) {
	receipt := &types.Receipt{
		TxHash: ctx.Tx.Hash,
	}

	// Add from nonce
	ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1)

	// Transfer amount
	amount, sender := ctx.Tx.Data.Amount, ctx.Tx.Data.From
	if ctx.Statedb.GetBalance(sender).Cmp(amount) < 0 {
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}

	ctx.Statedb.SubBalance(sender, amount)

	// check fee
	usedGas := ctx.Tx.IntrinsicGas()
	txFee := new(big.Int).Mul(ctx.Tx.Data.GasPrice, new(big.Int).SetUint64(usedGas))
	if ctx.Statedb.GetBalance(sender).Cmp(txFee) < 0 {
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}

	// handle fee
	ctx.Statedb.SubBalance(sender, txFee)
	minerFee := types.GetTxFeeShare(txFee)
	ctx.Statedb.AddBalance(ctx.BlockHeader.Creator, minerFee)

	return receipt, nil
}

func processSystemContract(ctx *Context, contract system.Contract, snapshot int) (*types.Receipt, error) {
	// must execute to make sure that system contract address is available
	if !ctx.Statedb.Exist(ctx.Tx.Data.To) {
		ctx.Statedb.CreateAccount(ctx.Tx.Data.To)
	}

	var err error
	receipt := &types.Receipt{
		TxHash: ctx.Tx.Hash,
	}

	// Add from nonce
	ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1)

	// Transfer amount
	amount, sender, recipient := ctx.Tx.Data.Amount, ctx.Tx.Data.From, ctx.Tx.Data.To
	if ctx.Statedb.GetBalance(sender).Cmp(amount) < 0 {
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}

	ctx.Statedb.SubBalance(sender, amount)
	ctx.Statedb.AddBalance(recipient, amount)

	// Run
	receipt.UsedGas = contract.RequiredGas(ctx.Tx.Data.Payload)
	receipt.Result, err = contract.Run(ctx.Tx.Data.Payload, system.NewContext(ctx.Tx, ctx.Statedb, ctx.BlockHeader))

	return receipt, err
}

func processEvmContract(ctx *Context, gas uint64) (*types.Receipt, error) {
	var err error
	receipt := &types.Receipt{
		TxHash: ctx.Tx.Hash,
	}

	statedb := &evm.StateDB{Statedb: ctx.Statedb}
	e := evm.NewEVMByDefaultConfig(ctx.Tx, statedb, ctx.BlockHeader, ctx.BcStore)
	caller := vm.AccountRef(ctx.Tx.Data.From)
	var leftOverGas uint64

	if ctx.Tx.Data.To.IsEmpty() {
		var createdContractAddr common.Address
		receipt.Result, createdContractAddr, leftOverGas, err = e.Create(caller, ctx.Tx.Data.Payload, gas, ctx.Tx.Data.Amount)
		if !createdContractAddr.IsEmpty() {
			receipt.ContractAddress = createdContractAddr.Bytes()
		}
	} else {
		ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1)
		receipt.Result, leftOverGas, err = e.Call(caller, ctx.Tx.Data.To, ctx.Tx.Data.Payload, gas, ctx.Tx.Data.Amount)
	}
	receipt.UsedGas = gas - leftOverGas

	return receipt, err
}

func handleFee(ctx *Context, receipt *types.Receipt, snapshot int) (*types.Receipt, error) {
	// Calculating the total fee
	// @todo decrease the gas fee
	usedGas := new(big.Int).SetUint64(receipt.UsedGas)
	totalFee := new(big.Int).Mul(usedGas, ctx.Tx.Data.GasPrice)

	// Calculating the From account balance is enough
	if balance := ctx.Statedb.GetBalance(ctx.Tx.Data.From); balance.Cmp(totalFee) < 0 {
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}

	// Transfer fee to coinbase
	ctx.Statedb.SubBalance(ctx.Tx.Data.From, totalFee)
	ctx.Statedb.AddBalance(ctx.BlockHeader.Creator, totalFee)
	receipt.TotalFee = totalFee.Uint64()

	// Record statedb hash
	var err error
	if receipt.PostState, err = ctx.Statedb.Hash(); err != nil {
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}

	// Add logs
	receipt.Logs = ctx.Statedb.GetCurrentLogs()
	if receipt.Logs == nil {
		receipt.Logs = make([]*types.Log, 0)
	}
	return receipt, nil
}

func revertStatedb(statedb *state.Statedb, snapshot int, err error) error {
	statedb.RevertToSnapshot(snapshot)
	return err
}
