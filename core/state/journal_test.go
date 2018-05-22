/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

func Test_Journal_CreateAccount(t *testing.T) {
	statedb, _, dispose := newTestEVMStateDB()
	defer dispose()

	snapshot := statedb.Snapshot()

	newAddr := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(newAddr)
	assert.Equal(t, statedb.getStateObject(newAddr), newStateObject(newAddr))

	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.getStateObject(newAddr), (*StateObject)(nil))
}

func Test_Journal_Balance(t *testing.T) {
	statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	stateObj.SetAmount(big.NewInt(100))

	// Sub balance
	snapshot := statedb.Snapshot()
	statedb.SubBalance(stateObj.address, big.NewInt(38))
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, stateObj.GetAmount(), big.NewInt(100))

	// Add balance
	snapshot = statedb.Snapshot()
	statedb.AddBalance(stateObj.address, big.NewInt(38))
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, stateObj.GetAmount(), big.NewInt(100))

	// Set balance
	snapshot = statedb.Snapshot()
	statedb.SetBalance(stateObj.address, big.NewInt(38))
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, stateObj.GetAmount(), big.NewInt(100))
}

func Test_Journal_Nonce(t *testing.T) {
	statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	stateObj.SetNonce(100)

	snapshot := statedb.Snapshot()
	statedb.SetNonce(stateObj.address, 38)
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, stateObj.GetNonce(), uint64(100))
}

func Test_Journal_Code(t *testing.T) {
	statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	snapshot := statedb.Snapshot()
	statedb.SetCode(stateObj.address, []byte("test code"))
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.GetCode(stateObj.address), []byte(nil))
}

func Test_Journal_Refund(t *testing.T) {
	statedb, _, dispose := newTestEVMStateDB()
	defer dispose()

	statedb.refund = 100

	snapshot := statedb.Snapshot()
	statedb.AddRefund(38)
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.GetRefund(), uint64(100))
}

func Test_Journal_State(t *testing.T) {
	statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	key := common.StringToHash("key")
	value := common.StringToHash("value")
	stateObj.cachedStorage[key] = value

	snapshot := statedb.Snapshot()
	statedb.SetState(stateObj.address, key, common.StringToHash("value2"))
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.GetState(stateObj.address, key), value)
}

func Test_Journal_Suicide(t *testing.T) {
	statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	snapshot := statedb.Snapshot()
	statedb.Suicide(stateObj.address)
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.HasSuicided(stateObj.address), false)
}
