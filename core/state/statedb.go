/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/hashicorp/golang-lru"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/trie"
)

const StateCacheCapacity = 1000

var (
	stateAmount0 = big.NewInt(0)
)

// Statedb use to store account with the MPT tee
type Statedb struct {
	trie         *trie.Trie
	stateObjects *lru.Cache // account address (common.Address) -> state object (*StateObject)
}

// NewStatedb new a statedb
func NewStatedb(root common.Hash, db database.Database) (*Statedb, error) {
	trie, err := trie.NewTrie(root, []byte("S"), db)
	if err != nil {
		return nil, err
	}

	stateCache, err := lru.New(StateCacheCapacity)
	if err != nil {
		return nil, err
	}

	return &Statedb{
		trie:         trie,
		stateObjects: stateCache,
	}, nil
}

// This is a memory copy of state db.
func (s *Statedb) GetCopy() *Statedb {
	copies, err := lru.New(StateCacheCapacity)
	if err != nil {
		panic(err) // only get err when StateCacheCapacity is negative, if so panic
	}

	for _, k := range s.stateObjects.Keys() {
		v, ok := s.stateObjects.Peek(k)
		if ok {
			copies.Add(k, v)
		}
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
	for _, key := range s.stateObjects.Keys() {
		value, ok := s.stateObjects.Peek(key)
		if ok {
			addr := key.(common.Address)
			object := value.(*StateObject)
			if object.dirty {
				err := s.commitOne(addr, object)
				if err != nil {
					return common.Hash{}, err
				}
				object.dirty = false
			}
		}
	}
	return s.trie.Commit(batch)
}

func (s *Statedb) commitOne(addr common.Address, obj *StateObject) error {
	data, err := rlp.EncodeToBytes(obj.account)
	if err != nil {
		return err
	}
	s.trie.Put(addr[:], data)

	return nil
}

// GetOrNewStateObject get or new a state object
func (s *Statedb) GetOrNewStateObject(addr common.Address) *StateObject {
	object := s.getStateObject(addr)
	if object == nil {
		object = newStateObject()
		object.SetNonce(0)
		if s.stateObjects.Len() == StateCacheCapacity {
			s.Commit(nil)
		}

		s.stateObjects.Add(addr, object)
	}

	return object
}

func (s *Statedb) getStateObject(addr common.Address) *StateObject {
	value, ok := s.stateObjects.Get(addr)
	if ok {
		object := value.(*StateObject)
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
	s.stateObjects.Add(addr, object)
	return object
}
