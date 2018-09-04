package evm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func Test_State(t *testing.T) {
	db, statedb, addr, dispose := newTestEVMStateDB()
	defer dispose()

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
	_, statedb2 := commitAndNewStateDB(db, statedb)

	// Ensure the state is valid.
	assert.Equal(t, statedb2.GetState(addr, k1), v1)
	assert.Equal(t, statedb2.GetState(addr, k2), v2)
}

func commitAndNewStateDB(db database.Database, statedb *StateDB) (common.Hash, *StateDB) {
	batch := db.NewBatch()
	rootHash, err := statedb.Commit(batch)
	if err != nil {
		panic(err)
	}

	if err = batch.Commit(); err != nil {
		panic(err)
	}

	newStatedb, err := state.NewStatedb(rootHash, db)
	if err != nil {
		panic(err)
	}

	return rootHash, &StateDB{newStatedb}
}

func newTestEVMStateDB() (database.Database, *StateDB, common.Address, func()) {
	db, dispose := leveldb.NewTestDatabase()

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		dispose()
		panic(err)
	}

	testAddr := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(testAddr)

	return db, &StateDB{statedb}, testAddr, dispose
}
