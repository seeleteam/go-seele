package state

import (
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/crypto"
)

func Test_AccountClone(t *testing.T) {
	// Account
	a1 := newTestAccount()

	// Function works well
	a2 := a1.clone()
	assert.Equal(t, a1, a2)

	// The change in src value cannot change the value of desc
	a1.Nonce = 2
	assert.Equal(t, a1.Nonce != a2.Nonce, true)
}

func Test_StateObjectClone(t *testing.T) {
	// StateObject
	addr := *crypto.MustGenerateRandomAddress()
	so1 := newStateObject(addr)
	so1.account = newTestAccount()

	// Function works well
	so2 := so1.GetCopy()
	assert.Equal(t, so1, so2)

	// The change in src value cannot change the value of desc
	so1.address = *crypto.MustGenerateRandomAddress()
	assert.Equal(t, so1.address != so2.address, true)
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
