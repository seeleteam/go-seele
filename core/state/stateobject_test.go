package state

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/stretchr/testify/assert"
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
	so2 := so1.clone()
	assert.Equal(t, so1, so2)

	// The change in src value cannot change the value of dest
	so1.address = *crypto.MustGenerateRandomAddress()
	so1.cachedStorage[common.StringToHash("hash1")] = []byte("value1")

	assert.Equal(t, so1.address != so2.address, true)
	assert.Equal(t, reflect.DeepEqual(so1.cachedStorage, so2.cachedStorage), false)
	assert.Equal(t, so1 != so2, true)
}

func newTestAccount() account {
	a1 := newAccount()
	a1.Amount = big.NewInt(100)
	a1.Nonce = 1
	a1.CodeHash = []byte("contract address")
	return a1
}

func newTestStateObject() *stateObject {
	addr := *crypto.MustGenerateRandomAddress()
	so1 := newStateObject(addr)
	so1.account = newTestAccount()
	so1.code = []byte("contract code")
	so1.dirtyCode = true
	so1.suicided = true
	so1.deleted = true

	return so1
}

func Test_StateObject_AmountDirty(t *testing.T) {
	so := newTestStateObject()

	// Nonce
	so.dirtyAccount = false
	nonce := uint64(101)
	so.setNonce(nonce)
	assert.Equal(t, so.getNonce(), nonce)
	assert.Equal(t, so.dirtyAccount, true)

	// Amount
	so.dirtyAccount = false
	amount := big.NewInt(101)
	so.setAmount(amount)
	assert.Equal(t, so.getAmount(), amount)
	assert.Equal(t, so.dirtyAccount, true)

	so.dirtyAccount = false
	so.subAmount(amount)
	assert.Equal(t, so.getAmount(), big.NewInt(0))
	assert.Equal(t, so.dirtyAccount, true)
}
