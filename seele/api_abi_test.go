/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package seele

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"

	"github.com/stretchr/testify/assert"
)

const (
	SimpleStorageABI  = "[{\"constant\":false,\"inputs\":[{\"name\":\"x\",\"type\":\"uint256\"}],\"name\":\"set\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"get\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"
	RemixGetPayload   = "0x6d4ce63c"
	RemixSet23Payload = "0x60fe47b10000000000000000000000000000000000000000000000000000000000000017"
)

func Test_GeneratePayload(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GeneratePayload")
	if common.FileOrFolderExists(dbPath) {
		os.RemoveAll(dbPath)
	}
	api := newTestAPI(t, dbPath)

	abiObj, err := abi.JSON(strings.NewReader(SimpleStorageABI))
	assert.NoError(t, err)

	// Get method test
	payload1, err1 := api.GeneratePayload(abiObj, "get")
	assert.NoError(t, err1)
	getPayload := hexutil.BytesToHex(payload1)
	assert.Equal(t, getPayload, RemixGetPayload)

	// Set method test
	payload2, err2 := api.GeneratePayload(abiObj, "set", big.NewInt(23))
	assert.NoError(t, err2)
	set23Payload := hexutil.BytesToHex(payload2)
	assert.Equal(t, set23Payload, RemixSet23Payload)

	// Invalid method test
	payload3, err3 := api.GeneratePayload(abiObj, "add", big.NewInt(23))
	assert.Error(t, err3)
	assert.Empty(t, payload3)

	// Invalid parameter type test
	payload4, err4 := api.GeneratePayload(abiObj, "set", 23)
	assert.Error(t, err4)
	assert.Empty(t, payload4)
}

func Test_GetAPI(t *testing.T) {
	dbPath := filepath.Join(common.GetTempFolder(), ".GetAPI")
	if common.FileOrFolderExists(dbPath) {
		os.RemoveAll(dbPath)
	}
	api := newTestAPI(t, dbPath)

	from, err := saveABI(api)
	assert.NoError(t, err)

	// Correctness test
	abiObj, err := abi.JSON(strings.NewReader(SimpleStorageABI))
	assert.NoError(t, err)
	abiObj1, err1 := api.GetABI(from)
	assert.NoError(t, err1)
	assert.Equal(t, abiObj1, abiObj)

	// Invalid address test
	abiObj2, err2 := api.GetABI(common.EmptyAddress)
	assert.Error(t, err2)
	assert.Equal(t, abiObj2, abi.ABI{})
}

func saveABI(api *PublicSeeleAPI) (common.Address, error) {
	statedb, err := api.s.chain.GetCurrentState()
	if err != nil {
		return common.EmptyAddress, err
	}

	from := getFromAddress(statedb)
	statedb.SetData(from, KeyABIHash, []byte(SimpleStorageABI))

	// save the statedb
	batch := api.s.accountStateDB.NewBatch()
	block := api.s.chain.CurrentBlock()
	block.Header.StateHash, _ = statedb.Commit(batch)
	block.Header.Height++
	block.Header.PreviousBlockHash = block.HeaderHash
	block.HeaderHash = block.Header.Hash()
	api.s.chain.GetStore().PutBlock(block, big.NewInt(1), true)
	batch.Commit()
	return from, nil
}
