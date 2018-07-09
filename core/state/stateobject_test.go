package state

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/crypto"
)

func Test_clone(t *testing.T) {
	// Account
	a1 := newAccount()
	a1.Nonce = 1
	a1.CodeHash = []byte("contract address")
	a1.StorageRootHash = []byte("root hash")

	// Function works well
	a2 := a1.clone()
	assert.Equal(t, a1.Amount, a2.Amount)
	assert.Equal(t, a1.Nonce, a2.Nonce)
	assert.Equal(t, a1.CodeHash, a2.CodeHash)
	assert.Equal(t, a1.StorageRootHash, a2.StorageRootHash)

	// The change in src value cannot change the value of desc
	a1.Nonce = 2
	if a1.Nonce == a2.Nonce {
		t.Fail()
	}

	// StateObject
	addr := *crypto.MustGenerateRandomAddress()
	sb1 := newStateObject(addr)

	// Function works well
	sb2 := sb1.GetCopy()
	assert.Equal(t, sb1.address, sb2.address)
	assert.Equal(t, sb1.addrHash, sb2.addrHash)
	assert.Equal(t, sb1.account, sb2.account)
	assert.Equal(t, sb1.dirtyAccount, sb2.dirtyAccount)
	assert.Equal(t, sb1.code, sb2.code)
	assert.Equal(t, sb1.dirtyCode, sb2.dirtyCode)
	assert.Equal(t, sb1.storageTrie, sb2.storageTrie)
	assert.Equal(t, sb1.cachedStorage, sb2.cachedStorage)
	assert.Equal(t, sb1.dirtyStorage, sb2.dirtyStorage)
	assert.Equal(t, sb1.suicided, sb2.suicided)
	assert.Equal(t, sb1.deleted, sb2.deleted)

	// The change in src value cannot change the value of desc
	sb1.address = *crypto.MustGenerateRandomAddress()
	if sb1.address == sb2.address {
		t.Fail()
	}
}
