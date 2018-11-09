package evm

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
)

// StateDB for evm
type StateDB struct {
	*state.Statedb
}

// GetState returns the value of the specified key in account storage if exists.
// Otherwise, return empty hash.
func (s *StateDB) GetState(address common.Address, key common.Hash) common.Hash {
	value := s.GetData(address, key)
	return common.BytesToHash(value)
}

// GetCommittedState returns the committed value of the specified key in account storage if exists.
func (s *StateDB) GetCommittedState(addr common.Address, hash common.Hash) common.Hash {
	value := s.GetCommittedData(addr, hash)
	return common.BytesToHash(value)
}

// SetState adds or updates the specified key-value pair in account storage.
func (s *StateDB) SetState(address common.Address, key common.Hash, value common.Hash) {
	s.SetData(address, key, value.Bytes())
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (s *StateDB) AddPreimage(common.Hash, []byte) {
	// Currently, do not support SHA3 preimage produced by EVM.
}

// ForEachStorage visits all the key-value pairs for the specified account storage.
func (s *StateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) {
	// do nothing, since ETH only call this method in test.
}
