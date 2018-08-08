/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/trie"
)

var (
	trieDbPrefix  = []byte("S")
	stateBalance0 = big.NewInt(0)
)

// Statedb is used to store accounts into the MPT tree
type Statedb struct {
	trie         *trie.Trie
	stateObjects map[common.Address]*stateObject

	dbErr  error  // dbErr is used for record the database error.
	refund uint64 // The refund counter, also used by state transitioning.

	// Receipt logs for current processed tx.
	curTxIndex uint
	curLogs    []*types.Log

	// State modifications for current processed tx.
	curJournal *journal
}

// NewStatedb constructs and returns a statedb instance
func NewStatedb(root common.Hash, db database.Database) (*Statedb, error) {
	trie, err := trie.NewTrie(root, trieDbPrefix, db)
	if err != nil {
		return nil, err
	}

	return &Statedb{
		trie:         trie,
		stateObjects: make(map[common.Address]*stateObject),
		curJournal:   newJournal(),
	}, nil
}

// GetCopy is a memory copy of state db.
func (s *Statedb) GetCopy() (*Statedb, error) {
	copyObjecsFunc := func(src map[common.Address]*stateObject) map[common.Address]*stateObject {
		dest := make(map[common.Address]*stateObject)
		for k, v := range src {
			dest[k] = v
		}
		return dest
	}

	cpyTrie, err := s.trie.ShallowCopy()
	if err != nil {
		return nil, err
	}

	return &Statedb{
		trie:         cpyTrie,
		stateObjects: copyObjecsFunc(s.stateObjects),

		dbErr:  s.dbErr,
		refund: s.refund,
	}, nil
}

// setError only records the first error.
func (s *Statedb) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

// GetBalance returns the balance of the specified account if exists.
// Otherwise, returns zero.
func (s *Statedb) GetBalance(addr common.Address) *big.Int {
	object := s.getStateObject(addr)
	if object != nil {
		return object.getAmount()
	}
	return stateBalance0
}

// SetBalance sets the balance of the specified account if exists.
func (s *Statedb) SetBalance(addr common.Address, balance *big.Int) {
	object := s.getStateObject(addr)
	if object != nil {
		s.curJournal.append(balanceChange{&addr, object.getAmount()})
		object.setAmount(balance)
	}
}

// AddBalance adds the specified amount to the balance for the specified account if exists.
func (s *Statedb) AddBalance(addr common.Address, amount *big.Int) {
	object := s.getStateObject(addr)
	if object != nil {
		s.curJournal.append(balanceChange{&addr, object.getAmount()})
		object.addAmount(amount)
	}
}

// SubBalance substracts the specified amount from the balance for the specified account if exists.
func (s *Statedb) SubBalance(addr common.Address, amount *big.Int) {
	object := s.getStateObject(addr)
	if object != nil {
		s.curJournal.append(balanceChange{&addr, object.getAmount()})
		object.subAmount(amount)
	}
}

// GetNonce gets the nonce of the specified account if exists.
// Otherwise, return 0.
func (s *Statedb) GetNonce(addr common.Address) uint64 {
	object := s.getStateObject(addr)
	if object != nil {
		return object.getNonce()
	}
	return 0
}

// SetNonce sets the nonce of the specified account if exists.
func (s *Statedb) SetNonce(addr common.Address, nonce uint64) {
	object := s.getStateObject(addr)
	if object != nil {
		s.curJournal.append(nonceChange{&addr, object.getNonce()})
		object.setNonce(nonce)
	}
}

// Hash flush the dirty data into trie and calculates the intermediate root hash.
func (s *Statedb) Hash() (common.Hash, error) {
	if s.dbErr != nil {
		return common.EmptyHash, s.dbErr
	}

	for addr := range s.curJournal.dirties {
		if object, found := s.stateObjects[addr]; found {
			if err := object.flush(s.trie); err != nil {
				return common.EmptyHash, err
			}
		}
	}

	s.clearJournalAndRefund()

	return s.trie.Hash(), nil
}

// Commit persists the trie to the specified batch.
func (s *Statedb) Commit(batch database.Batch) (common.Hash, error) {
	if batch == nil {
		panic("batch is nil")
	}

	if s.dbErr != nil {
		return common.EmptyHash, s.dbErr
	}

	for _, object := range s.stateObjects {
		if err := object.flush(s.trie); err != nil {
			return common.EmptyHash, err
		}
	}

	return s.trie.Commit(batch), nil
}

func (s *Statedb) getStateObject(addr common.Address) *stateObject {
	// get from cache
	if object, ok := s.stateObjects[addr]; ok {
		if !object.deleted {
			return object
		}

		// object has already been deleted from trie.
		return nil
	}

	// load from trie
	object := newStateObject(addr)
	if ok, err := object.loadAccount(s.trie); !ok || err != nil {
		return nil
	}

	// add in cache
	s.stateObjects[addr] = object

	return object
}

// Prepare resets the logs and journal to process a new tx.
func (s *Statedb) Prepare(txIndex int) {
	s.curTxIndex = uint(txIndex)
	s.curLogs = nil

	s.clearJournalAndRefund()
}

func (s *Statedb) clearJournalAndRefund() {
	s.refund = 0
	s.curJournal.entries = s.curJournal.entries[:0]
	s.curJournal.dirties = make(map[common.Address]uint)
}

// GetCurrentLogs returns the current transaction logs.
func (s *Statedb) GetCurrentLogs() []*types.Log {
	return s.curLogs
}
