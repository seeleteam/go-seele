package evm

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
)

// StateDB for evm
type StateDB struct {
	*state.Statedb
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (s *StateDB) AddPreimage(common.Hash, []byte) {
	// Currently, do not support SHA3 preimage produced by EVM.
}

// ForEachStorage visits all the key-value pairs for the specified account storage.
func (s *StateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) {
	// do nothing, since ETH only call this method in test.
}
