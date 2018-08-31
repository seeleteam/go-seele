/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package svm

import (
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/svm/evm"
	"github.com/seeleteam/go-seele/core/svm/native"
	"github.com/seeleteam/go-seele/core/types"
)

// SeeleVM is heterogeneous and adaptive
type SeeleVM interface {
	Process(tx *types.Transaction, txIndex int) (*types.Receipt, error)
}

// Context for other vm constructs
type Context struct {
	Tx          *types.Transaction
	Statedb     *state.Statedb
	BlockHeader *types.BlockHeader
	BcStore     store.BlockchainStore
}

// NewSeeleVM creates a Seele VM with specified context to process transaction
func NewSeeleVM(ctx *Context) SeeleVM {
	// System contract process
	if ctx.Tx.Data.To.IsReserved() {
		if contract := system.GetContractByAddress(ctx.Tx.Data.To); contract != nil {
			return native.NewNativeVM(ctx.Tx, ctx.Statedb, ctx.BlockHeader, ctx.BcStore, contract)
		}
	}

	// EVM
	statedb := &evm.StateDB{Statedb: ctx.Statedb}
	return &evm.EVM{Evm: evm.NewEVMByDefaultConfig(ctx.Tx, statedb, ctx.BlockHeader, ctx.BcStore)}
}
