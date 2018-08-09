/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

// CreateAccount creates a new account in statedb.
func (s *Statedb) CreateAccount(address common.Address) {
	stateObj := s.getStateObject(address)
	if stateObj != nil {
		return
	}

	stateObj = newStateObject(address)
	s.curJournal.append(createObjectChange{&address})
	s.stateObjects[address] = stateObj
}

// GetCodeHash returns the hash of the contract code associated with the specified address if any.
// Otherwise, return an empty hash.
func (s *Statedb) GetCodeHash(address common.Address) common.Hash {
	stateObj := s.getStateObject(address)
	if stateObj == nil {
		return common.EmptyHash
	}

	return common.BytesToHash(stateObj.account.CodeHash)
}

// GetCode returns the contract code associated with the specified address if any.
// Otherwise, return nil.
func (s *Statedb) GetCode(address common.Address) []byte {
	stateObj := s.getStateObject(address)
	if stateObj == nil {
		return nil
	}

	return stateObj.loadCode(s.trie)
}

// SetCode sets the contract code of the specified address if exists.
func (s *Statedb) SetCode(address common.Address, code []byte) {
	// EVM call SetCode after CreateAccount during contract creation.
	// So, here the retrieved stateObj should not be nil.
	stateObj := s.getStateObject(address)
	if stateObj == nil {
		return
	}

	prevCode := stateObj.loadCode(s.trie)
	s.curJournal.append(codeChange{&address, prevCode})
	stateObj.setCode(code)
}

// GetCodeSize returns the size of the contract code associated with the specified address if any.
// Otherwise, return 0.
func (s *Statedb) GetCodeSize(address common.Address) int {
	code := s.GetCode(address)
	return len(code)
}

// AddRefund refunds the specified gas value
func (s *Statedb) AddRefund(gas uint64) {
	s.curJournal.append(refundChange{s.refund})
	s.refund += gas
}

// GetRefund returns the current value of the refund counter.
func (s *Statedb) GetRefund() uint64 {
	return s.refund
}

// GetState returns the value of the specified key in account storage if exists.
// Otherwise, return empty hash.
func (s *Statedb) GetState(address common.Address, key common.Hash) common.Hash {
	stateObj := s.getStateObject(address)
	if stateObj == nil {
		return common.EmptyHash
	}

	return stateObj.getState(s.trie, key)
}

// SetState adds or updates the specified key-value pair in account storage.
func (s *Statedb) SetState(address common.Address, key common.Hash, value common.Hash) {
	stateObj := s.getStateObject(address)
	if stateObj == nil {
		return
	}

	prevValue := stateObj.getState(s.trie, key)
	s.curJournal.append(storageChange{&address, key, prevValue})
	stateObj.setState(key, value)
}

// Suicide marks the given account as suicided and clears the account balance.
// Note the account's state object is still available until the state is committed.
// Return true if the specified account exists, otherwise false.
func (s *Statedb) Suicide(address common.Address) bool {
	stateObj := s.getStateObject(address)
	if stateObj == nil {
		return false
	}

	s.curJournal.append(suicideChange{&address, stateObj.suicided, stateObj.getAmount()})

	stateObj.setAmount(new(big.Int))
	stateObj.suicided = true

	return true
}

// HasSuicided returns true if the specified account exists and suicided, otherwise false.
func (s *Statedb) HasSuicided(address common.Address) bool {
	stateObj := s.getStateObject(address)
	if stateObj == nil {
		return false
	}

	return stateObj.suicided
}

// Exist indicates whether the given account exists in statedb.
// Note that it should also return true for suicided accounts.
func (s *Statedb) Exist(address common.Address) bool {
	return s.getStateObject(address) != nil
}

// Empty indicates whether the given account satisfies (balance = nonce = code = 0).
func (s *Statedb) Empty(address common.Address) bool {
	stateObj := s.getStateObject(address)
	return stateObj == nil || stateObj.empty()
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (s *Statedb) RevertToSnapshot(revid int) {
	s.curJournal.revert(s, revid)
}

// Snapshot returns an identifier for the current revision of the statedb.
func (s *Statedb) Snapshot() int {
	return s.curJournal.snapshot()
}

// AddLog adds a log.
func (s *Statedb) AddLog(log *types.Log) {
	log.TxIndex = s.curTxIndex

	s.curLogs = append(s.curLogs, log)
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (s *Statedb) AddPreimage(common.Hash, []byte) {
	// Currently, do not support SHA3 preimage produced by EVM.
}

// ForEachStorage visits all the key-value pairs for the specified account storage.
func (s *Statedb) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) {
	// do nothing, since ETH only call this method in test.
}
