/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"bytes"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
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
	address  common.Address
	addrHash common.Hash

	account      account
	dirtyAccount bool

	code      []byte // contract code
	dirtyCode bool

	cachedStorage map[common.Hash][]byte // cache the retrieved account states.
	dirtyStorage  map[common.Hash][]byte // changed account states that need to flush to DB.

	// When a state object is marked as suicided, it will be deleted from the trie when commit the state DB.
	suicided bool

	// When a state object is marked as deleted, need not to load from trie again.
	deleted bool
}

func newStateObject(address common.Address) *stateObject {
	return &stateObject{
		address:       address,
		addrHash:      crypto.MustHash(address),
		account:       newAccount(),
		dirtyAccount:  true,
		cachedStorage: make(map[common.Hash][]byte),
		dirtyStorage:  make(map[common.Hash][]byte),
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

func copyStorage(src map[common.Hash][]byte) map[common.Hash][]byte {
	cloned := make(map[common.Hash][]byte)

	for k, v := range src {
		cloned[k] = common.CopyBytes(v)
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

// subAmount subtracts the specified amount from the balance of the account in the state object
func (s *stateObject) subAmount(amount *big.Int) {
	s.setAmount(new(big.Int).Sub(s.account.Amount, amount))
}

func (s *stateObject) dataKey(dataType byte, prefix ...byte) []byte {
	key := append(s.addrHash.Bytes(), dataType)
	return append(key, prefix...)
}

func (s *stateObject) loadAccount(trie Trie) (bool, error) {
	value, ok, err := trie.Get(s.dataKey(dataTypeAccount))
	if err != nil || !ok {
		return false, err
	}

	if err := common.Deserialize(value, &s.account); err != nil {
		return false, err
	}

	return true, nil
}

func (s *stateObject) loadCode(trie Trie) ([]byte, error) {
	// already loaded
	if s.code != nil {
		return s.code, nil
	}

	// no code
	if len(s.account.CodeHash) == 0 {
		return nil, nil
	}

	// load code from trie
	code, ok, err := trie.Get(s.dataKey(dataTypeCode))
	if err != nil || !ok {
		return nil, err
	}

	s.code = code

	return code, nil
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

func (s *stateObject) setState(key common.Hash, value []byte) {
	s.dirtyStorage[key] = value
}

func (s *stateObject) getState(trie Trie, key common.Hash, committed bool) ([]byte, error) {
	if !committed {
		if value, ok := s.dirtyStorage[key]; ok {
			return value, nil
		}
	}

	if value, ok := s.cachedStorage[key]; ok {
		return value, nil
	}

	value, ok, err := trie.Get(s.dataKey(dataTypeStorage, crypto.MustHash(key).Bytes()...))
	if err != nil || !ok {
		return nil, err
	}

	s.cachedStorage[key] = value

	return value, nil
}

// flush update the dirty data of state object to the specified trie if any.
func (s *stateObject) flush(trie Trie) error {
	// Flush storage change.
	if len(s.dirtyStorage) > 0 {
		for k, v := range s.dirtyStorage {
			// value cached and not changed.
			if cachedValue, ok := s.cachedStorage[k]; ok && bytes.Equal(cachedValue, v) {
				continue
			}

			s.cachedStorage[k] = v

			if err := trie.Put(s.dataKey(dataTypeStorage, crypto.MustHash(k).Bytes()...), v); err != nil {
				return err
			}
		}
		s.dirtyStorage = make(map[common.Hash][]byte)
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
		trie.DeletePrefix(s.addrHash.Bytes())
		s.deleted = true
	}

	return nil
}
