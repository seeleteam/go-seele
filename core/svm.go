/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/core/vm"
)

// SVMTYPE for vm type
type SVMTYPE string

const (
	// EVM is ethereum virtual machine
	EVM SVMTYPE = "evm"
)

// SeeleVM is heterogeneous and adaptive
type SeeleVM interface {
	Process(tx *types.Transaction, txIndex int) (*types.Receipt, error)
}

// NewSeeleVM implements a variety of vm, and you must ensure that the SVMTYPE is completed, otherwise the returns result is nil
func NewSeeleVM(svmType SVMTYPE, tx *types.Transaction, statedb StateDB, blockHeader *types.BlockHeader, bcStore store.BlockchainStore) SeeleVM {
	var svm SeeleVM
	switch svmType {
	case EVM:
		return vm.NewEVMByDefaultConfig(tx, statedb, blockHeader, bcStore)
	}
	return svm
}

// StateDB is an SVM database for full state querying.
type StateDB interface {
	CreateAccount(common.Address)

	SubBalance(common.Address, *big.Int)
	AddBalance(common.Address, *big.Int)
	GetBalance(common.Address) *big.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	SetCode(common.Address, []byte)
	GetCodeSize(common.Address) int

	AddRefund(uint64)
	GetRefund() uint64

	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash)

	Suicide(common.Address) bool
	HasSuicided(common.Address) bool

	// Exist reports whether the given account exists in state.
	// Notably this should also return true for suicided accounts.
	Exist(common.Address) bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(common.Address) bool

	RevertToSnapshot(int)
	Snapshot() int

	AddLog(*types.Log)
	AddPreimage(common.Hash, []byte)

	ForEachStorage(common.Address, func(common.Hash, common.Hash) bool)
}
