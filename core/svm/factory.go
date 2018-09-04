package svm

import (
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/svm/evm"
	"github.com/seeleteam/go-seele/core/svm/native"
)

// Type for svm
type Type int

// SVM Type
const (
	EVM Type = iota
	Native
)

// CreateSVM creates a Seele VM with specified context to process transaction
func CreateSVM(ctx *Context, _type Type) SeeleVM {
	switch _type {
	case Native:
		contract := system.GetContractByAddress(ctx.Tx.Data.To)
		return native.NewNativeVM(ctx.Tx, ctx.Statedb, ctx.BlockHeader, ctx.BcStore, contract)
	case EVM:
		statedb := &evm.StateDB{Statedb: ctx.Statedb}
		return &evm.EVM{Evm: evm.NewEVMByDefaultConfig(ctx.Tx, statedb, ctx.BlockHeader, ctx.BcStore)}
	default:
		return CreateSVM(ctx, EVM)
	}
}
