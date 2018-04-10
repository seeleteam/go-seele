/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/trie"
)

var (
	stateAmount0 = big.NewInt(0)
)

// Statedb use to store account with the MPT tee
type Statedb struct {
	trie         *trie.Trie
	stateObjects map[common.Address]*StateObject // add LRU for this?
}

// NewStatedb new a statedb
func NewStatedb(root common.Hash, db database.Database) (*Statedb, error) {
	trie, err := trie.NewTrie(root, []byte("S"), db)
	if err != nil {
		return nil, err
	}
	return &Statedb{
		trie:         trie,
		stateObjects: make(map[common.Address]*StateObject),
	}, nil
}

// This is a memory copy of state db.
func (s *Statedb) GetCopy() *Statedb {
	copies := make(map[common.Address]*StateObject)
	for k, v := range s.stateObjects {
		copies[k] = v.GetCopy()
	}

	return &Statedb{
		trie:         s.trie,
		stateObjects: copies,
	}
}

// GetAmount get amount of account
func (s *Statedb) GetAmount(addr common.Address) (*big.Int, bool) {
	object := s.getStateObject(addr)
	if object != nil {
		return object.GetAmount(), true
	}
	return stateAmount0, false
}

// SetAmount set amount of account
func (s *Statedb) SetAmount(addr common.Address, amount *big.Int) {
	object := s.getStateObject(addr)
	if object != nil {
		object.SetAmount(amount)
	}
}

// AddAmount add amount for account
func (s *Statedb) AddAmount(addr common.Address, amount *big.Int) {
	object := s.getStateObject(addr)
	if object != nil {
		object.AddAmount(amount)
	}
}

// SubAmount sub amount for account
func (s *Statedb) SubAmount(addr common.Address, amount *big.Int) {
	object := s.getStateObject(addr)
	if object != nil {
		object.SubAmount(amount)
	}
}

// GetNonce get nonce of account
func (s *Statedb) GetNonce(addr common.Address) (uint64, bool) {
	object := s.getStateObject(addr)
	if object != nil {
		return object.GetNonce(), true
	}
	return 0, false
}

// SetNonce set nonce of account
func (s *Statedb) SetNonce(addr common.Address, nonce uint64) {
	object := s.getStateObject(addr)
	if object != nil {
		object.SetNonce(nonce)
	}
}

// Commit commit memory state object to db
func (s *Statedb) Commit(batch database.Batch) (root common.Hash, err error) {
	for addr, object := range s.stateObjects {
		if object.dirty {
			data, err := rlp.EncodeToBytes(object.account)
			if err != nil {
				return common.Hash{}, err
			}
			s.trie.Put(addr[:], data)
			object.dirty = false
		}
	}
	return s.trie.Commit(batch)
}

// GetOrNewStateObject get or new a state object
func (s *Statedb) GetOrNewStateObject(addr common.Address) *StateObject {
	object := s.getStateObject(addr)
	if object == nil {
		object = newStateObject()
		object.SetNonce(0)
		s.stateObjects[addr] = object
	}
	return object
}

func (s *Statedb) getStateObject(addr common.Address) *StateObject {
	if object := s.stateObjects[addr]; object != nil {
		return object
	}
	object := newStateObject()
	val, _ := s.trie.Get(addr[:])
	if len(val) == 0 {
		return nil
	}
	if err := rlp.DecodeBytes(val, &object.account); err != nil {
		return nil
	}
	s.stateObjects[addr] = object
	return object
}
