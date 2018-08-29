/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package svm

import (
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/svm/evm"
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

// NewSeeleVM implements a variety of vm, and you must ensure that the SVMTYPE is completed, otherwise the returns result is nil
func NewSeeleVM(ctx *Context) SeeleVM {
	// TODO for other vm
	return &evm.EVM{
		Evm: evm.NewEVMByDefaultConfig(ctx.Tx, ctx.Statedb, ctx.BlockHeader, ctx.BcStore),
	}
}
