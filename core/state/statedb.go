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
	if object := s.getStateObject(addr); object != nil {
		return object.getAmount()
	}

	return stateBalance0
}

// SetBalance sets the balance of the specified account if exists.
func (s *Statedb) SetBalance(addr common.Address, balance *big.Int) {
	if object := s.getStateObject(addr); object != nil {
		s.curJournal.append(balanceChange{&addr, object.getAmount()})
		object.setAmount(balance)
	}
}

// AddBalance adds the specified amount to the balance for the specified account if exists.
func (s *Statedb) AddBalance(addr common.Address, amount *big.Int) {
	if object := s.getStateObject(addr); object != nil {
		s.curJournal.append(balanceChange{&addr, object.getAmount()})
		object.addAmount(amount)
	}
}

// SubBalance substracts the specified amount from the balance for the specified account if exists.
func (s *Statedb) SubBalance(addr common.Address, amount *big.Int) {
	if object := s.getStateObject(addr); object != nil {
		s.curJournal.append(balanceChange{&addr, object.getAmount()})
		object.subAmount(amount)
	}
}

// GetNonce gets the nonce of the specified account if exists.
// Otherwise, return 0.
func (s *Statedb) GetNonce(addr common.Address) uint64 {
	if object := s.getStateObject(addr); object != nil {
		return object.getNonce()
	}

	return 0
}

// SetNonce sets the nonce of the specified account if exists.
func (s *Statedb) SetNonce(addr common.Address, nonce uint64) {
	if object := s.getStateObject(addr); object != nil {
		s.curJournal.append(nonceChange{&addr, object.getNonce()})
		object.setNonce(nonce)
	}
}

// GetData returns the account data of the specified key if exists.
// Otherwise, return nil.
func (s *Statedb) GetData(addr common.Address, key common.Hash) []byte {
	if object := s.getStateObject(addr); object != nil {
		return object.getState(s.trie, key)
	}

	return nil
}

// SetData sets the key value pair for the specified account if exists.
func (s *Statedb) SetData(addr common.Address, key common.Hash, value []byte) {
	if object := s.getStateObject(addr); object != nil {
		prevValue := object.getState(s.trie, key)
		s.curJournal.append(storageChange{&addr, key, prevValue})
		object.setState(key, value)
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

// Prepare resets the logs and journal to process a new tx and return the statedb snapshot.
func (s *Statedb) Prepare(txIndex int) int {
	s.curTxIndex = uint(txIndex)
	s.curLogs = nil

	s.clearJournalAndRefund()
	return s.Snapshot()
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

// CreateAccount creates a new account in statedb.
func (s *Statedb) CreateAccount(address common.Address) {
	if object := s.getStateObject(address); object == nil {
		object = newStateObject(address)
		s.curJournal.append(createObjectChange{&address})
		s.stateObjects[address] = object
	}
}

// GetCodeHash returns the hash of the contract code associated with the specified address if any.
// Otherwise, return an empty hash.
func (s *Statedb) GetCodeHash(address common.Address) common.Hash {
	if object := s.getStateObject(address); object != nil {
		return common.BytesToHash(object.account.CodeHash)
	}

	return common.EmptyHash
}

// GetCode returns the contract code associated with the specified address if any.
// Otherwise, return nil.
func (s *Statedb) GetCode(address common.Address) []byte {
	if object := s.getStateObject(address); object != nil {
		return object.loadCode(s.trie)
	}

	return nil
}

// SetCode sets the contract code of the specified address if exists.
func (s *Statedb) SetCode(address common.Address, code []byte) {
	// EVM call SetCode after CreateAccount during contract creation.
	// So, here the retrieved stateObj should not be nil.
	if object := s.getStateObject(address); object != nil {
		prevCode := object.loadCode(s.trie)
		s.curJournal.append(codeChange{&address, prevCode})
		object.setCode(code)
	}
}

// GetCodeSize returns the size of the contract code associated with the specified address if any.
// Otherwise, return 0.
func (s *Statedb) GetCodeSize(address common.Address) int {
	code := s.GetCode(address)
	return len(code)
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

// AddRefund refunds the specified gas value
func (s *Statedb) AddRefund(gas uint64) {
	s.curJournal.append(refundChange{s.refund})
	s.refund += gas
}

// GetRefund returns the current value of the refund counter.
func (s *Statedb) GetRefund() uint64 {
	return s.refund
}
