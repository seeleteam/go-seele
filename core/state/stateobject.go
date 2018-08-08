/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/trie"
)

var (
	dataTypeAccount = byte('0')
	dataTypeCode    = byte('1')
	dataTypeStorage = byte('2')
)

// account is a balance model for blockchain
type account struct {
	Nonce    uint64
	Amount   *big.Int
	CodeHash []byte // contract code hash
}

func newAccount() account {
	return account{
		Amount: new(big.Int),
	}
}

func (a account) clone() account {
	return account{
		Nonce:    a.Nonce,
		Amount:   new(big.Int).Set(a.Amount),
		CodeHash: common.CopyBytes(a.CodeHash),
	}
}

// stateObject is the state object for statedb
type stateObject struct {
	address common.Address

	account      account
	dirtyAccount bool

	code      []byte // contract code
	dirtyCode bool

	cachedStorage map[common.Hash]common.Hash // cache the retrieved account states.
	dirtyStorage  map[common.Hash]common.Hash // changed account states that need to flush to DB.

	// When a state object is marked as suicided, it will be deleted from the trie when commit the state DB.
	suicided bool

	// When a state object is marked as deleted, need not to load from trie again.
	deleted bool
}

func newStateObject(address common.Address) *stateObject {
	return &stateObject{
		address:       address,
		account:       newAccount(),
		dirtyAccount:  true,
		cachedStorage: make(map[common.Hash]common.Hash),
		dirtyStorage:  make(map[common.Hash]common.Hash),
	}
}

func (s *stateObject) clone() *stateObject {
	cloned := *s

	cloned.account = s.account.clone()
	cloned.code = common.CopyBytes(s.code)
	cloned.cachedStorage = copyStorage(s.cachedStorage)
	cloned.dirtyStorage = copyStorage(s.dirtyStorage)

	return &cloned
}

func copyStorage(src map[common.Hash]common.Hash) map[common.Hash]common.Hash {
	cloned := make(map[common.Hash]common.Hash)

	for k, v := range src {
		cloned[k] = v
	}

	return cloned
}

// setNonce sets the nonce of the account in the state object
func (s *stateObject) setNonce(nonce uint64) {
	s.account.Nonce = nonce
	s.dirtyAccount = true
}

// getNonce gets the nonce of the account in the state object
func (s *stateObject) getNonce() uint64 {
	return s.account.Nonce
}

// getAmount gets the balance amount of the account in the state object
func (s *stateObject) getAmount() *big.Int {
	return new(big.Int).Set(s.account.Amount)
}

// setAmount sets the balance amount of the account in the state object
func (s *stateObject) setAmount(amount *big.Int) {
	if amount.Sign() >= 0 {
		s.account.Amount.Set(amount)
		s.dirtyAccount = true
	}
}

// addAmount adds the specified amount to the balance of the account in the state object
func (s *stateObject) addAmount(amount *big.Int) {
	s.setAmount(new(big.Int).Add(s.account.Amount, amount))
}

// subAmount substracts the specified amount from the balance of the account in the state object
func (s *stateObject) subAmount(amount *big.Int) {
	s.setAmount(new(big.Int).Sub(s.account.Amount, amount))
}

func (s *stateObject) dataKey(dataType byte, prefix ...byte) []byte {
	key := append(s.address.Bytes(), dataType)
	return append(key, prefix...)
}

func (s *stateObject) loadAccount(trie *trie.Trie) (bool, error) {
	value, ok := trie.Get(s.dataKey(dataTypeAccount))
	if !ok {
		return false, nil
	}

	if err := common.Deserialize(value, &s.account); err != nil {
		return false, err
	}

	return true, nil
}

func (s *stateObject) loadCode(trie *trie.Trie) []byte {
	// already loaded
	if s.code != nil {
		return s.code
	}

	// no code
	if len(s.account.CodeHash) == 0 {
		return nil
	}

	// load code from trie
	code, ok := trie.Get(s.dataKey(dataTypeCode))
	if !ok {
		return nil
	}

	s.code = code

	return code
}

func (s *stateObject) setCode(code []byte) {
	s.code = code
	s.dirtyCode = true

	if len(code) == 0 {
		s.account.CodeHash = nil
	} else {
		s.account.CodeHash = crypto.HashBytes(code).Bytes()
	}
	s.dirtyAccount = true
}

// empty returns whether the account is considered empty (nonce == amount == 0 and no code).
// This is used during EVM execution.
func (s *stateObject) empty() bool {
	return s.account.Nonce == 0 && s.account.Amount.Sign() == 0 && len(s.account.CodeHash) == 0
}

func (s *stateObject) setState(key, value common.Hash) {
	s.cachedStorage[key] = value
	s.dirtyStorage[key] = value
}

func (s *stateObject) getState(trie *trie.Trie, key common.Hash) common.Hash {
	if value, ok := s.cachedStorage[key]; ok {
		return value
	}

	if value, ok := trie.Get(s.dataKey(dataTypeStorage, key.Bytes()...)); ok {
		return common.BytesToHash(value)
	}

	return common.EmptyHash
}

// flush update the dirty data of state object to the specified trie if any.
func (s *stateObject) flush(trie *trie.Trie) error {
	// Flush storage change.
	if len(s.dirtyStorage) > 0 {
		for k, v := range s.dirtyStorage {
			if err := trie.Put(s.dataKey(dataTypeStorage, k.Bytes()...), v.Bytes()); err != nil {
				return err
			}
		}
		s.dirtyStorage = make(map[common.Hash]common.Hash)
	}

	// Flush code change.
	if s.dirtyCode {
		if err := trie.Put(s.dataKey(dataTypeCode), s.code); err != nil {
			return err
		}

		s.dirtyCode = false
	}

	// Flush account info change.
	if s.dirtyAccount {
		if err := trie.Put(s.dataKey(dataTypeAccount), common.SerializePanic(s.account)); err != nil {
			return err
		}
		s.dirtyAccount = false
	}

	// Remove the account from state DB if suicided.
	if s.suicided && !s.deleted {
		trie.DeletePrefix(s.address.Bytes())
		s.deleted = true
	}

	return nil
}
