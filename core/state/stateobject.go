/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/trie"
)

var (
	keyPrefixCode   = []byte("code")
	dbPrefixStorage = []byte("s")
)

// Account is a balance model for blockchain
type Account struct {
	Nonce           uint64
	Amount          *big.Int
	CodeHash        common.Hash // contract code hash
	StorageRootHash common.Hash // merkle root of the storage trie
}

// StateObject is the state object for statedb
type StateObject struct {
	address  common.Address
	addrHash common.Hash

	account      Account
	dirtyAccount bool

	code      []byte // contract code
	dirtyCode bool

	storageTrie   *trie.Trie
	cachedStorage map[common.Hash]common.Hash // cache the retrieved account states.
	dirtyStorage  map[common.Hash]common.Hash // changed account states that need to flush to DB.

	// When a state object is marked assuicided, it will be deleted from the trie when commit the state DB.
	suicided bool
}

func newStateObject(address common.Address) *StateObject {
	return &StateObject{
		address:  address,
		addrHash: crypto.HashBytes(address.Bytes()),
		account: Account{
			Amount: new(big.Int),
		},
		cachedStorage: make(map[common.Hash]common.Hash),
		dirtyStorage:  make(map[common.Hash]common.Hash),
	}
}

// GetCopy gets a copy of the state object
func (s *StateObject) GetCopy() *StateObject {
	codeCloned := make([]byte, len(s.code))
	copy(codeCloned, s.code)

	objCloned := *s
	objCloned.account.Amount = big.NewInt(0).Set(s.account.Amount)
	objCloned.code = codeCloned

	return &objCloned
}

// SetNonce sets the nonce of the account in the state object
func (s *StateObject) SetNonce(nonce uint64) {
	s.account.Nonce = nonce
	s.dirtyAccount = true
}

// GetNonce gets the nonce of the account in the state object
func (s *StateObject) GetNonce() uint64 {
	return s.account.Nonce
}

// GetAmount gets the balance amount of the account in the state object
func (s *StateObject) GetAmount() *big.Int {
	return new(big.Int).Set(s.account.Amount)
}

// SetAmount sets the balance amount of the account in the state object
func (s *StateObject) SetAmount(amount *big.Int) {
	if amount.Sign() >= 0 {
		s.account.Amount.Set(amount)
		s.dirtyAccount = true
	}
}

// AddAmount adds the specified amount to the balance of the account in the state object
func (s *StateObject) AddAmount(amount *big.Int) {
	s.SetAmount(new(big.Int).Add(s.account.Amount, amount))
}

// SubAmount substracts the specified amount from the balance of the account in the state object
func (s *StateObject) SubAmount(amount *big.Int) {
	s.SetAmount(new(big.Int).Sub(s.account.Amount, amount))
}

func (s *StateObject) loadCode(db database.Database) ([]byte, error) {
	if s.code != nil {
		return s.code, nil
	}

	if s.account.CodeHash.IsEmpty() {
		return nil, nil
	}

	code, err := db.Get(s.getCodeKey())
	if err != nil {
		return nil, err
	}

	s.code = code

	return code, nil
}

func (s *StateObject) getCodeKey() []byte {
	return append(keyPrefixCode, s.addrHash.Bytes()...)
}

func (s *StateObject) setCode(code []byte) {
	s.code = code
	s.dirtyCode = true

	s.account.CodeHash = crypto.HashBytes(code)
	s.dirtyAccount = true
}

func (s *StateObject) serializeCode(batch database.Batch) {
	if s.code != nil {
		batch.Put(s.getCodeKey(), s.code)
	}
}

// empty returns whether the account is considered empty (nonce == amount == 0 and no code).
// This is used during EVM execution.
func (s *StateObject) empty() bool {
	return s.account.Nonce == 0 && s.account.Amount.Sign() == 0 && s.account.CodeHash.IsEmpty()
}

func (s *StateObject) setState(key, value common.Hash) {
	s.cachedStorage[key] = value

	if old, ok := s.dirtyStorage[key]; !ok || !old.Equal(value) {
		s.dirtyStorage[key] = value
	}
}

func (s *StateObject) getState(db database.Database, key common.Hash) (common.Hash, error) {
	if value, ok := s.cachedStorage[key]; ok {
		return value, nil
	}

	if err := s.ensureStorageTrie(db); err != nil {
		return common.EmptyHash, err
	}

	if value, ok := s.storageTrie.Get(s.getStorageKey(key)); ok {
		return common.BytesToHash(value), nil
	}

	return common.EmptyHash, nil
}

func (s *StateObject) ensureStorageTrie(db database.Database) error {
	if s.storageTrie != nil {
		return nil
	}

	trie, err := trie.NewTrie(s.account.StorageRootHash, dbPrefixStorage, db)
	if err != nil {
		return err
	}

	s.storageTrie = trie

	return nil
}

func (s *StateObject) getStorageKey(key common.Hash) []byte {
	// trie key: address hash + storage key
	return append(s.addrHash.Bytes(), key.Bytes()...)
}

// commitStorageTrie flush dirty storage to trie if any, and update the storage merkle root hash.
func (s *StateObject) commitStorageTrie(trieDB database.Database, commitBatch database.Batch) error {
	if len(s.dirtyStorage) == 0 {
		return nil
	}

	if err := s.ensureStorageTrie(trieDB); err != nil {
		return err
	}

	for k, v := range s.dirtyStorage {
		if err := s.storageTrie.Put(s.getStorageKey(k), v.Bytes()); err != nil {
			return err
		}
	}

	// Update the storage merkle root hash and mark account as dirty.
	s.account.StorageRootHash = s.storageTrie.Commit(commitBatch)
	s.dirtyAccount = true

	// Reset dirty storage flag
	s.dirtyStorage = make(map[common.Hash]common.Hash)

	return nil
}
