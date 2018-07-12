/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func newTestEVMStateDB() (*Statedb, *StateObject, func()) {
	db, dispose := leveldb.NewTestDatabase()

	statedb, err := NewStatedb(common.EmptyHash, db)
	if err != nil {
		dispose()
		panic(err)
	}

	testAddr := *crypto.MustGenerateRandomAddress()
	statedb.GetOrNewStateObject(testAddr)

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

func commitAndNewStateDB(statedb *Statedb) (common.Hash, *Statedb) {
	batch := statedb.db.NewBatch()
	rootHash, err := statedb.Commit(batch)
	if err != nil {
		panic(err)
	}

	if err = batch.Commit(); err != nil {
		panic(err)
	}

	newStatedb, err := NewStatedb(rootHash, statedb.db)
	if err != nil {
		panic(err)
	}

	return rootHash, newStatedb
}

func Test_CreateAccount(t *testing.T) {
	statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	// Assert the no code for a new created account.
	assert.Equal(t, statedb.dbErr, error(nil))
	assert.Equal(t, stateObj.account.CodeHash, []byte(nil))
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

	// Commit the account code change and create another state DB with the same root hash.
	_, statedb2 := commitAndNewStateDB(statedb)
	assert.Equal(t, stateObj.dirtyCode, false)

	// Ensure the account code is valid.
	assert.Equal(t, statedb2.GetCodeHash(addr), codeHash)
	assert.Equal(t, statedb2.GetCode(addr), code)
	assert.Equal(t, statedb2.GetCodeSize(addr), len(code))
	assert.Equal(t, stateObj.dirtyCode, false)

	// Empty address
	var emptyCode []byte
	addr = common.EmptyAddress
	assert.Equal(t, statedb2.GetCodeHash(addr), common.EmptyHash)
	assert.Equal(t, statedb2.GetCode(addr), emptyCode)
	assert.Equal(t, statedb2.GetCodeSize(addr), 0)
	assert.Equal(t, statedb2.Empty(addr), true)

	// An address that does not exist
	addr = *crypto.MustGenerateRandomAddress()
	assert.Equal(t, statedb2.GetCodeHash(addr), common.EmptyHash)
	assert.Equal(t, statedb2.GetCode(addr), emptyCode)
	assert.Equal(t, statedb2.GetCodeSize(addr), 0)
	assert.Equal(t, statedb2.Empty(addr), true)
}

func Test_Refund(t *testing.T) {
	statedb, _, dispose := newTestEVMStateDB()
	defer dispose()

	assert.Equal(t, statedb.refund, uint64(0))

	statedb.AddRefund(38)
	assert.Equal(t, statedb.refund, uint64(38))

	statedb.AddRefund(66)
	assert.Equal(t, statedb.refund, uint64(38+66))
}

func Test_State(t *testing.T) {
	statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	addr := stateObj.address
	k1 := common.StringToHash("key1")
	v1 := common.StringToHash("value1")

	// By default, no state in account.
	assert.Equal(t, statedb.GetState(addr, k1), common.EmptyHash)

	// Set k1-v1
	statedb.SetState(addr, k1, v1)
	assert.Equal(t, statedb.GetState(addr, k1), v1)

	// Set k2-v2
	k2 := common.StringToHash("key2")
	v2 := common.StringToHash("value2")
	statedb.SetState(addr, k2, v2)
	assert.Equal(t, statedb.GetState(addr, k2), v2)

	// Commit the state change
	_, statedb2 := commitAndNewStateDB(statedb)

	// Ensure the state is valid.
	assert.Equal(t, statedb2.GetState(addr, k1), v1)
	assert.Equal(t, statedb2.GetState(addr, k2), v2)
}

func Test_Suicide(t *testing.T) {
	statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	addr := stateObj.address

	assert.Equal(t, statedb.Exist(addr), true)
	assert.Equal(t, statedb.HasSuicided(addr), false)

	assert.Equal(t, statedb.Suicide(*crypto.MustGenerateRandomAddress()), false)
	assert.Equal(t, statedb.Suicide(addr), true)
	assert.Equal(t, statedb.HasSuicided(addr), true)

	// Commit the state change
	_, statedb2 := commitAndNewStateDB(statedb)

	// Ensure the account does not exist.
	assert.Equal(t, statedb2.Exist(addr), false)
}

func Test_Log(t *testing.T) {
	statedb, _, dispose := newTestEVMStateDB()
	defer dispose()

	statedb.Prepare(38)

	statedb.AddLog(new(types.Log))
	statedb.AddLog(new(types.Log))
	statedb.AddLog(new(types.Log))

	logs := statedb.GetCurrentLogs()
	assert.Equal(t, len(logs), 3)
	assert.Equal(t, logs[0].TxIndex, uint(38))
	assert.Equal(t, logs[1].TxIndex, uint(38))
	assert.Equal(t, logs[2].TxIndex, uint(38))
}
