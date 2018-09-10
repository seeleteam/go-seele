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
	var err error
	var receipt *types.Receipt
	snapshot := ctx.Statedb.Prepare(ctx.TxIndex)

	if contract := system.GetContractByAddress(ctx.Tx.Data.To); contract != nil { // system contract
		receipt, err = processSystemContract(ctx, contract, snapshot)
	} else if ctx.Tx.IsCrossShardTx() && !ctx.Tx.Data.To.IsEVMContract() {
		return processCrossShardTransaction(ctx, snapshot)
	} else { // evm
		receipt, err = processEvmContract(ctx)
	}

	// Gas is not enough
	if err == vm.ErrInsufficientBalance {
		return nil, err
	}

	if err != nil {
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
	if ctx.Statedb.GetBalance(sender).Cmp(ctx.Tx.Data.Fee) < 0 {
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}

	// handle fee
	ctx.Statedb.SubBalance(sender, ctx.Tx.Data.Fee)
	txFee := types.GetTxFeeShare(ctx.Tx.Data.Fee)
	ctx.Statedb.AddBalance(ctx.BlockHeader.Creator, txFee)

	return receipt, nil
}

func processSystemContract(ctx *Context, contract system.Contract, snapshot int) (*types.Receipt, error) {
	// must execute to make sure that system contract account is available
	system.Create(ctx.Statedb)
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

func processEvmContract(ctx *Context) (*types.Receipt, error) {
	var err error
	receipt := &types.Receipt{
		TxHash: ctx.Tx.Hash,
	}

	statedb := &evm.StateDB{Statedb: ctx.Statedb}
	e := evm.NewEVMByDefaultConfig(ctx.Tx, statedb, ctx.BlockHeader, ctx.BcStore)
	caller := vm.AccountRef(ctx.Tx.Data.From)
	gas, leftOverGas := maxTxGas, uint64(0)

	// Currently, use maxTxGas gas to bypass ErrInsufficientBalance error and avoid overly complex contract creation or calculation.
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
	gasFee := big.NewInt(0)
	if ctx.Tx.Data.To.IsEmpty() {
		gasFee = contractCreationFee(ctx.Tx.Data.Payload)
	} else {
		gasFee = usedGasFee(receipt.UsedGas)
	}
	totalFee := big.NewInt(0).Add(gasFee, ctx.Tx.Data.Fee)

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
