/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package state

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/stretchr/testify/assert"
)

func Test_Journal_CreateAccount(t *testing.T) {
	_, statedb, _, dispose := newTestEVMStateDB()
	defer dispose()

	snapshot := statedb.Snapshot()

	newAddr := *crypto.MustGenerateRandomAddress()
	statedb.CreateAccount(newAddr)
	assert.Equal(t, statedb.getStateObject(newAddr), newStateObject(newAddr))

	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.getStateObject(newAddr), (*stateObject)(nil))
}

func Test_Journal_Balance(t *testing.T) {
	_, statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	stateObj.setAmount(big.NewInt(100))

	// Sub balance
	snapshot := statedb.Snapshot()
	statedb.SubBalance(stateObj.address, big.NewInt(38))
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, stateObj.getAmount(), big.NewInt(100))

	// Add balance
	snapshot = statedb.Snapshot()
	statedb.AddBalance(stateObj.address, big.NewInt(38))
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, stateObj.getAmount(), big.NewInt(100))

	// Set balance
	snapshot = statedb.Snapshot()
	statedb.SetBalance(stateObj.address, big.NewInt(38))
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, stateObj.getAmount(), big.NewInt(100))
}

func Test_Journal_Nonce(t *testing.T) {
	_, statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	stateObj.setNonce(100)

	snapshot := statedb.Snapshot()
	statedb.SetNonce(stateObj.address, 38)
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, stateObj.getNonce(), uint64(100))
}

func Test_Journal_Code(t *testing.T) {
	_, statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	snapshot := statedb.Snapshot()
	statedb.SetCode(stateObj.address, []byte("test code"))
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.GetCode(stateObj.address), []byte(nil))
}

func Test_Journal_Refund(t *testing.T) {
	_, statedb, _, dispose := newTestEVMStateDB()
	defer dispose()

	statedb.refund = 100

	snapshot := statedb.Snapshot()
	statedb.AddRefund(38)
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.GetRefund(), uint64(100))
}

func Test_Journal_State(t *testing.T) {
	_, statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	key := common.StringToHash("key")
	value := common.StringToHash("value")
	stateObj.cachedStorage[key] = value.Bytes()

	snapshot := statedb.Snapshot()
	statedb.SetData(stateObj.address, key, []byte("value2"))
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.GetData(stateObj.address, key), value.Bytes())
}

func Test_Journal_Suicide(t *testing.T) {
	_, statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	stateObj.setAmount(big.NewInt(100))

	snapshot := statedb.Snapshot()
	statedb.Suicide(stateObj.address)
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.HasSuicided(stateObj.address), false)
	assert.Equal(t, statedb.GetBalance(stateObj.address), big.NewInt(100))

	// Suicide a state object that already suicided.
	stateObj.suicided = true
	snapshot = statedb.Snapshot()
	statedb.Suicide(stateObj.address)
	statedb.RevertToSnapshot(snapshot)
	assert.Equal(t, statedb.HasSuicided(stateObj.address), true)
	assert.Equal(t, statedb.GetBalance(stateObj.address), big.NewInt(100))
}

func Test_Journal_MultipleSnapshot(t *testing.T) {
	_, statedb, stateObj, dispose := newTestEVMStateDB()
	defer dispose()

	statedb.clearJournalAndRefund()

	// Snapshot 1: 2 changes
	snapshot1 := statedb.Snapshot()
	assert.Equal(t, snapshot1, 0)
	statedb.SetNonce(stateObj.address, 5)
	statedb.AddBalance(stateObj.address, big.NewInt(38))

	// Snapshot 2: 2 changes
	snapshot2 := statedb.Snapshot()
	assert.Equal(t, snapshot2, 2) // 2 changes in snapshot 1
	statedb.SetNonce(stateObj.address, 10)
	statedb.AddBalance(stateObj.address, big.NewInt(62))

	// Check the state object
	assert.Equal(t, statedb.GetNonce(stateObj.address), uint64(10))
	assert.Equal(t, statedb.GetBalance(stateObj.address), big.NewInt(100))

	// Revert snapshot 2
	statedb.RevertToSnapshot(snapshot2)
	assert.Equal(t, statedb.GetNonce(stateObj.address), uint64(5))
	assert.Equal(t, statedb.GetBalance(stateObj.address), big.NewInt(38))

	// Revert snapshot 1
	statedb.RevertToSnapshot(snapshot1)
	assert.Equal(t, statedb.GetNonce(stateObj.address), uint64(0))
	assert.Equal(t, statedb.GetBalance(stateObj.address), big.NewInt(0))

	// Check journal info
	assert.Equal(t, statedb.Snapshot(), 0)
}
