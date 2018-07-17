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
	CodeHash        []byte // contract code hash
	StorageRootHash []byte // merkle root of the storage trie
}

func newAccount() Account {
	return Account{
		Amount: new(big.Int),
	}
}

func (a Account) clone() Account {
	return Account{
		Nonce:           a.Nonce,
		Amount:          new(big.Int).Set(a.Amount),
		CodeHash:        common.CopyBytes(a.CodeHash),
		StorageRootHash: common.CopyBytes(a.StorageRootHash),
	}
}

// StateObject is the state object for statedb
type StateObject struct {
	address  common.Address
	addrHash common.Hash

	account      Account
	dirtyAccount bool

	code      []byte // contract code
	dirtyCode bool

	storageTrie      *trie.Trie
	cachedStorage    map[common.Hash]common.Hash // cache the retrieved account states.
	dirtyStorage     map[common.Hash]common.Hash // changed account states that need to flush to DB.
	storageTrieDirty bool

	// When a state object is marked as suicided, it will be deleted from the trie when commit the state DB.
	suicided bool

	// When a state object is marked as deleted, need not to load from trie again.
	deleted bool
}

func newStateObject(address common.Address) *StateObject {
	return &StateObject{
		address:       address,
		addrHash:      crypto.HashBytes(address.Bytes()),
		account:       newAccount(),
		dirtyAccount:  true,
		cachedStorage: make(map[common.Hash]common.Hash),
		dirtyStorage:  make(map[common.Hash]common.Hash),
	}
}

// GetCopy gets a copy of the state object
func (s *StateObject) GetCopy() *StateObject {
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

	if len(s.account.CodeHash) == 0 {
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

	if len(code) == 0 {
		s.account.CodeHash = nil
	} else {
		s.account.CodeHash = crypto.HashBytes(code).Bytes()
	}
	s.dirtyAccount = true
}

func (s *StateObject) serializeCode(batch database.Batch) {
	if len(s.code) > 0 {
		batch.Put(s.getCodeKey(), s.code)
	}
}

// empty returns whether the account is considered empty (nonce == amount == 0 and no code).
// This is used during EVM execution.
func (s *StateObject) empty() bool {
	return s.account.Nonce == 0 && s.account.Amount.Sign() == 0 && len(s.account.CodeHash) == 0
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

	rootHash := common.EmptyHash
	if len(s.account.StorageRootHash) > 0 {
		rootHash = common.BytesToHash(s.account.StorageRootHash)
	}

	trie, err := trie.NewTrie(rootHash, dbPrefixStorage, db)
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
	// not dirty and do not persist storage trie
	if len(s.dirtyStorage) == 0 && (commitBatch == nil || !s.storageTrieDirty) {
		return nil
	}

	if err := s.ensureStorageTrie(trieDB); err != nil {
		return err
	}

	// flush dirty storage into trie if any dirty
	for k, v := range s.dirtyStorage {
		if err := s.storageTrie.Put(s.getStorageKey(k), v.Bytes()); err != nil {
			return err
		}

		s.storageTrieDirty = true
	}

	// set account as dirty if dirty and reset dirty storage cache
	if len(s.dirtyStorage) > 0 {
		s.dirtyAccount = true
		s.dirtyStorage = make(map[common.Hash]common.Hash)
	}

	// commit to update the storage root hash or persist the storage trie to DB
	s.account.StorageRootHash = s.storageTrie.Commit(commitBatch).Bytes()

	if commitBatch != nil {
		s.storageTrieDirty = false
	}

	return nil
}

// commit flush dirty data of state object to the specified db and trie if any.
func (s *StateObject) commit(storageTrieDB database.Database, trie *trie.Trie, batch database.Batch) error {
	// Commit storage change.
	if err := s.commitStorageTrie(storageTrieDB, batch); err != nil {
		return err
	}

	// Commit code change.
	if s.dirtyCode && batch != nil {
		s.serializeCode(batch)
		s.dirtyCode = false
	}

	// Commit account info change.
	if s.dirtyAccount {
		data := common.SerializePanic(s.account)
		trie.Put(s.address.Bytes(), data)
		s.dirtyAccount = false
	}

	// Remove the account from state DB if suicided.
	if s.suicided && !s.deleted {
		s.deleted = true
		trie.Delete(s.address.Bytes())
	}

	return nil
}
