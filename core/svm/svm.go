/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package svm

import (
	"math/big"

	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/svm/evm"
	"github.com/seeleteam/go-seele/core/svm/native"
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
	var receipt *types.Receipt
	var err error
	var IsHandledNonceAndAmount bool

	snapshot := ctx.Statedb.Snapshot()
	ctx.Statedb.Prepare(ctx.TxIndex)

	if contract := system.GetContractByAddress(ctx.Tx.Data.To); contract != nil { // system contract
		vm := native.NewNativeVM(ctx.Tx, ctx.Statedb, ctx.BlockHeader, ctx.BcStore, contract)
		receipt, err = vm.ProcessTransaction(ctx.Tx)
	} else {
		statedb := &evm.StateDB{Statedb: ctx.Statedb}
		vm := &evm.EVM{Evm: evm.NewEVMByDefaultConfig(ctx.Tx, statedb, ctx.BlockHeader, ctx.BcStore)}
		receipt, err = vm.ProcessTransaction(ctx.Tx)
		IsHandledNonceAndAmount = true
	}

	// ProcessTransaction
	if err != nil {
		ctx.Statedb.RevertToSnapshot(snapshot)
		return nil, err
	}

	// Calculating the From account balance is enough
	totalFee := new(big.Int).SetUint64(receipt.TotalFee)
	if balance := ctx.Statedb.GetBalance(ctx.Tx.Data.From); balance.Cmp(totalFee) < 0 {
		ctx.Statedb.RevertToSnapshot(snapshot)
		return nil, vm.ErrInsufficientBalance
	}

	// Transfer fee to coinbase
	ctx.Statedb.SubBalance(ctx.Tx.Data.From, totalFee)
	ctx.Statedb.AddBalance(ctx.BlockHeader.Creator, totalFee)

	if !IsHandledNonceAndAmount {
		// Transfer amount
		amount, sender, recipient := ctx.Tx.Data.Amount, ctx.Tx.Data.From, ctx.Tx.Data.To
		if ctx.Statedb.GetBalance(sender).Cmp(amount) < 0 {
			ctx.Statedb.RevertToSnapshot(snapshot)
			return nil, vm.ErrInsufficientBalance
		}

		ctx.Statedb.SubBalance(sender, amount)
		ctx.Statedb.AddBalance(recipient, amount)

		// Add from nonce
		ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1)
	}

	// Record statedb hash
	if receipt.PostState, err = ctx.Statedb.Hash(); err != nil {
		ctx.Statedb.RevertToSnapshot(snapshot)
		return nil, err
	}

	// Add logs
	receipt.Logs = ctx.Statedb.GetCurrentLogs()
	if receipt.Logs == nil {
		receipt.Logs = make([]*types.Log, 0)
	}
	return receipt, nil
}
