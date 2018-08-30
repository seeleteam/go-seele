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

// NewSeeleVM implements a variety of vm
func NewSeeleVM(ctx *Context) SeeleVM {
	// NVM
	_, ok := system.Contracts[ctx.Tx.Data.To]
	if ok {
		return native.NewNativeVM(ctx.Tx, ctx.Statedb, ctx.BlockHeader, ctx.BcStore)
	}

	// EVM
	statedb := &evm.StateDB{Statedb: ctx.Statedb}
	return &evm.EVM{Evm: evm.NewEVMByDefaultConfig(ctx.Tx, statedb, ctx.BlockHeader, ctx.BcStore)}
}
