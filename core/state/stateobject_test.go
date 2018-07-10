package state

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/trie"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/crypto"
)

func Test_AccountClone(t *testing.T) {
	// Account
	a1 := newTestAccount()

	// Function works well
	a2 := a1.clone()
	assert.Equal(t, a1, a2)

	// The change in src value cannot change the value of dest
	a1.Nonce = 2
	assert.Equal(t, a1.Nonce != a2.Nonce, true)
}

func Test_StateObjectClone(t *testing.T) {
	// StateObject
	so1 := newTestStateObject()

	// Function works well
	so2 := so1.GetCopy()
	assert.Equal(t, so1, so2)

	// The change in src value cannot change the value of dest
	so1.address = *crypto.MustGenerateRandomAddress()
	hash1 := common.BytesToHash([]byte("root hash1"))
	so1.cachedStorage[hash1] = hash1

	assert.Equal(t, so1.address != so2.address, true)
	assert.Equal(t, reflect.DeepEqual(so1.cachedStorage, so2.cachedStorage), false)
	assert.Equal(t, so1 != so2, true)
}

func newTestAccount() Account {
	a1 := newAccount()
	a1.Amount = big.NewInt(100)
	a1.Nonce = 1
	a1.CodeHash = []byte("contract address")
	a1.StorageRootHash = []byte("root hash")
	return a1
}

func newTestStateObject() *StateObject {
	addr := *crypto.MustGenerateRandomAddress()
	so1 := newStateObject(addr)
	so1.account = newTestAccount()
	so1.code = []byte("contract code")
	so1.dirtyCode = true
	so1.suicided = true
	so1.deleted = true

	db, remove := leveldb.NewTestDatabase()
	defer remove()
	so1.storageTrie, _ = trie.NewTrie(common.EmptyHash, []byte("dbprefix"), db)
	so1.cachedStorage[common.BytesToHash(so1.account.StorageRootHash)] = common.BytesToHash(so1.account.StorageRootHash)
	return so1
}

func Test_StateObject(t *testing.T) {
	so := newTestStateObject()

	// Nonce
	so.dirtyAccount = false
	nonce := uint64(101)
	so.SetNonce(nonce)
	assert.Equal(t, so.GetNonce(), nonce)
	assert.Equal(t, so.dirtyAccount, true)

	// Ammount
	so.dirtyAccount = false
	ammount := big.NewInt(101)
	so.SetAmount(ammount)
	assert.Equal(t, so.GetAmount(), ammount)
	assert.Equal(t, so.dirtyAccount, true)

	so.dirtyAccount = false
	so.SubAmount(ammount)
	assert.Equal(t, so.GetAmount(), big.NewInt(0))
	assert.Equal(t, so.dirtyAccount, true)
}
