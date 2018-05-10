/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

func newTestEVMStateDB() (*Statedb, *StateObject, func()) {
	db, dispose := newTestStateDB()

	statedb, err := NewStatedb(common.EmptyHash, db)
	if err != nil {
		dispose()
		panic(err)
	}

	testAddr := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(testAddr)

	stateObj := statedb.getStateObject(testAddr)
	if stateObj == nil {
		dispose()
		panic("cannot find the state object.")
	}

	if !stateObj.address.Equal(testAddr) {
		dispose()
		panic("the address of state object is invalid.")
	}

	return statedb, stateObj, dispose
}

func Test_CreateAccount(t *testing.T) {
	_, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	// Assert the no code for a new created account.
	assert.Equal(t, stateObj.dbErr, error(nil))
	assert.Equal(t, stateObj.account.CodeHash, common.EmptyHash)
	assert.Equal(t, stateObj.code, []byte(nil))
	assert.Equal(t, stateObj.dirtyCode, false)
}

func Test_Code(t *testing.T) {
	statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	addr := stateObj.address

	// Set code to account
	code := []byte("test code")
	codeHash := crypto.HashBytes(code)
	statedb.SetCode(addr, code)

	// Validate code APIs
	assert.Equal(t, statedb.GetCodeHash(addr), codeHash)
	assert.Equal(t, statedb.GetCode(addr), code)
	assert.Equal(t, statedb.GetCodeSize(addr), len(code))
	assert.Equal(t, stateObj.dirtyCode, true)

	// Commit the account code change
	batch := statedb.db.NewBatch()
	rootHash := statedb.Commit(batch)
	assert.Equal(t, batch.Commit(), error(nil))
	assert.Equal(t, stateObj.dirtyCode, false)

	// Create another state DB with the same root hash.
	statedb2, err := NewStatedb(rootHash, statedb.db)
	if err != nil {
		panic(err)
	}

	// Ensure the account code is valid.
	assert.Equal(t, statedb2.GetCodeHash(addr), codeHash)
	assert.Equal(t, statedb2.GetCode(addr), code)
	assert.Equal(t, statedb2.GetCodeSize(addr), len(code))
	assert.Equal(t, stateObj.dirtyCode, false)
}
