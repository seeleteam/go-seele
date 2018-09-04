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
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/core/vm"
)

// SeeleVM is heterogeneous and adaptive
type SeeleVM interface {
	ProcessTransaction(tx *types.Transaction) (*types.Receipt, error)
}

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
	_type := EVM
	if ctx.Tx.Data.To.IsReserved() {
		if contract := system.GetContractByAddress(ctx.Tx.Data.To); contract != nil {
			_type = Native
		}
	}
	s := CreateSVM(ctx, _type)

	ctx.Statedb.Prepare(ctx.TxIndex)

	// ProcessTransaction
	receipt, err := s.ProcessTransaction(ctx.Tx)
	if err != nil {
		return nil, err
	}

	// Calculating the From account balance is enough
	totalFee := new(big.Int).SetUint64(receipt.TotalFee)
	if balance := ctx.Statedb.GetBalance(ctx.Tx.Data.From); balance.Cmp(totalFee) < 0 {
		return nil, vm.ErrInsufficientBalance
	}

	// Transfer fee to coinbase
	ctx.Statedb.SubBalance(ctx.Tx.Data.From, totalFee)
	ctx.Statedb.AddBalance(ctx.BlockHeader.Creator, totalFee)

	if _type != EVM {
		// Transfer amount
		amount, sender, recipient := ctx.Tx.Data.Amount, ctx.Tx.Data.From, ctx.Tx.Data.To
		if ctx.Statedb.GetBalance(sender).Cmp(amount) < 0 {
			return nil, vm.ErrInsufficientBalance
		}

		ctx.Statedb.SubBalance(sender, amount)

		if recipient.IsEmpty() {
			recipient = common.BytesToAddress(receipt.ContractAddress)
		}
		ctx.Statedb.AddBalance(recipient, amount)

		// Add from nonce
		ctx.Statedb.SetNonce(ctx.Tx.Data.From, ctx.Tx.Data.AccountNonce+1)
	}

	// Record statedb hash
	if receipt.PostState, err = ctx.Statedb.Hash(); err != nil {
		return nil, err
	}

	// Add logs
	receipt.Logs = ctx.Statedb.GetCurrentLogs()
	if receipt.Logs == nil {
		receipt.Logs = make([]*types.Log, 0)
	}
	return receipt, nil
}
