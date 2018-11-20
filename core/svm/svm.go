/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package svm

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
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
	// check the tx against the latest statedb, e.g. balance, nonce.
	if err := ctx.Tx.ValidateState(ctx.Statedb); err != nil {
		return nil, errors.NewStackedError(err, "failed to validate tx against statedb")
	}

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
		receipt, err = processSystemContract(ctx, contract, snapshot, leftOverGas)
	} else if ctx.Tx.IsCrossShardTx() && !ctx.Tx.Data.To.IsEVMContract() { // cross shard tx
		return processCrossShardTransaction(ctx, snapshot)
	} else { // evm
		receipt, err = processEvmContract(ctx, leftOverGas)
	}

	// account balance is not enough (account.balance < tx.amount)
	if err == vm.ErrInsufficientBalance {
		return nil, revertStatedb(ctx.Statedb, snapshot, err)
	}

	if err != nil {
		ctx.Statedb.RevertToSnapshot(snapshot)
		receipt.Failed = true
		receipt.Result = []byte(err.Error())
	}

	// include the intrinsic gas
	receipt.UsedGas += intrGas

	// refund gas, capped to half of the used gas.
	refund := ctx.Statedb.GetRefund()
	if maxRefund := receipt.UsedGas / 2; refund > maxRefund {
		refund = maxRefund
	}
	receipt.UsedGas -= refund

	return handleFee(ctx, receipt, snapshot)
}

func processCrossShardTransaction(ctx *Context, snapshot int) (*types.Receipt, error) {
	receipt := &types.Receipt{
		TxHash:  ctx.Tx.Hash,
		UsedGas: types.TransferAmountIntrinsicGas * 2,
	}

	// Add from nonce
	ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1)

	// Transfer amount
	amount, sender := ctx.Tx.Data.Amount, ctx.Tx.Data.From
	if ctx.Statedb.GetBalance(sender).Cmp(amount) < 0 {
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}

	ctx.Statedb.SubBalance(sender, amount)

	// check fee, only support non-contract tx.
	txFee := new(big.Int).Mul(ctx.Tx.Data.GasPrice, new(big.Int).SetUint64(receipt.UsedGas))
	if ctx.Statedb.GetBalance(sender).Cmp(txFee) < 0 {
		return nil, revertStatedb(ctx.Statedb, snapshot, vm.ErrInsufficientBalance)
	}

	// handle fee
	ctx.Statedb.SubBalance(sender, txFee)
	minerFee := types.GetTxFeeShare(txFee)
	ctx.Statedb.AddBalance(ctx.BlockHeader.Creator, minerFee)

	return receipt, nil
}

func processSystemContract(ctx *Context, contract system.Contract, snapshot int, leftOverGas uint64) (*types.Receipt, error) {
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

	// Check used gas is over flow
	receipt.UsedGas = contract.RequiredGas(ctx.Tx.Data.Payload)
	if receipt.UsedGas > leftOverGas {
		return receipt, vm.ErrOutOfGas
	}
	// Run
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
	// For normal tx: fee = 20k * 1 Fan/gas = 0.0002 Seele
	// For contract tx, average gas per tx is about 100k on ETH, fee = 100k * 1Fan/gas = 0.001 Seele
	usedGas := new(big.Int).SetUint64(receipt.UsedGas)
	totalFee := new(big.Int).Mul(usedGas, ctx.Tx.Data.GasPrice)

	// Transfer fee to coinbase
	// Note, the sender should always have enough balance.
	ctx.Statedb.SubBalance(ctx.Tx.Data.From, totalFee)
	ctx.Statedb.AddBalance(ctx.BlockHeader.Creator, totalFee)
	receipt.TotalFee = totalFee.Uint64()

	// Record statedb hash
	var err error
	if receipt.PostState, err = ctx.Statedb.Hash(); err != nil {
		err = errors.NewStackedError(err, "failed to get statedb root hash")
		return nil, revertStatedb(ctx.Statedb, snapshot, err)
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
