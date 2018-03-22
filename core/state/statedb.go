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

// Statedb use to store accout with the merkle trie
type Statedb struct {
	trie         *trie.Trie
	stateObjects map[common.Address]*stateObject
}

// NewStatedb new a statedb
func NewStatedb(root common.Hash, db database.Database) (*Statedb, error) {
	trie, err := trie.NewTrie(root, []byte("S"), db)
	if err != nil {
		return nil, err
	}
	return &Statedb{
		trie:         trie,
		stateObjects: make(map[common.Address]*stateObject),
	}, nil
}

// ResetStatedb reset state db
func (s *Statedb) ResetStatedb(root common.Hash, db database.Database) error {
	trie, err := trie.NewTrie(root, []byte("S"), db)
	if err != nil {
		return err
	}
	s.trie = trie
	s.stateObjects = make(map[common.Address]*stateObject)
	return nil
}

// GetAmount get amount of account
func (s *Statedb) GetAmount(addr common.Address) *big.Int {
	object := s.getStateObject(addr)
	if object != nil {
		return object.GetAmount()
	}
	return big.NewInt(0)
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
func (s *Statedb) GetNonce(addr common.Address) uint64 {
	object := s.getStateObject(addr)
	if object != nil {
		return object.GetNonce()
	}
	return 0
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
		}
	}
	return s.trie.Commit(batch)
}

func (s *Statedb) getStateObject(addr common.Address) *stateObject {
	if object := s.stateObjects[addr]; object != nil {
		return object
	}
	object := newStateObject()
	val, _ := s.trie.Get(addr[:])
	if len(val) == 0 {
		object.SetNonce(0)
		s.stateObjects[addr] = object
		return object
	}
	if err := rlp.DecodeBytes(val, &object.account); err != nil {
		return nil
	}
	s.stateObjects[addr] = object
	return object
}
