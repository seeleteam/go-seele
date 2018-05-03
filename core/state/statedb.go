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

// StateCacheCapacity state cache capacity
const StateCacheCapacity = 1000

var (
	stateBalance0 = big.NewInt(0)
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

	cpyTrie, err := s.trie.ShallowCopyTrie()
	if err != nil {
		return nil, err
	}

	return &Statedb{
		trie:         cpyTrie,
		stateObjects: copies,
	}, nil
}

// GetBalance returns the balance of specified account if exists.
// Otherwise, returns zero.
func (s *Statedb) GetBalance(addr common.Address) *big.Int {
	object := s.getStateObject(addr)
	if object != nil {
		return object.GetAmount()
	}
	return stateBalance0
}

// SetBalance set the balance of specified account.
func (s *Statedb) SetBalance(addr common.Address, balance *big.Int) {
	object := s.getStateObject(addr)
	if object != nil {
		object.SetAmount(balance)
	}
}

// AddBalance add balance for account
func (s *Statedb) AddBalance(addr common.Address, amount *big.Int) {
	object := s.getStateObject(addr)
	if object != nil {
		object.AddAmount(amount)
	}
}

// SubBalance sub amount for account
func (s *Statedb) SubBalance(addr common.Address, amount *big.Int) {
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
func (s *Statedb) Commit(batch database.Batch) common.Hash {
	for _, key := range s.stateObjects.Keys() {
		value, ok := s.stateObjects.Peek(key)
		if ok {
			addr := key.(common.Address)
			object := value.(*StateObject)
			if object.dirty {
				s.commitOne(addr, object)
				object.dirty = false
			}
		}
	}
	return s.trie.Commit(batch)
}

func (s *Statedb) commitOne(addr common.Address, obj *StateObject) {
	data, err := rlp.EncodeToBytes(obj.account)
	if err != nil {
		panic(err) // must encode because object account is a deterministic struct
	}
	s.trie.Put(addr[:], data)
}

func (s *Statedb) cache(addr common.Address, obj *StateObject) {
	if s.stateObjects.Len() == StateCacheCapacity {
		s.Commit(nil)

		// clear a quarter of the cached state info to avoid frequency commit
		for i := 0; i < StateCacheCapacity/4; i++ {
			s.stateObjects.RemoveOldest()
		}
	}

	s.stateObjects.Add(addr, obj)
}

// GetOrNewStateObject get or new a state object
func (s *Statedb) GetOrNewStateObject(addr common.Address) *StateObject {
	object := s.getStateObject(addr)
	if object == nil {
		object = newStateObject()
		object.SetNonce(0)
		s.cache(addr, object)
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
	s.cache(addr, object)
	return object
}
