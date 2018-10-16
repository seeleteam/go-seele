/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package seele

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seeleteam/go-seele/common"

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
	api, args := newTestAPI(t, dbPath), make([]string, 0)

	// Get method test
	payload1, err1 := api.GeneratePayload(SimpleStorageABI, "get", args)
	assert.NoError(t, err1)
	assert.Equal(t, payload1, RemixGetPayload)

	// Set method test
	args = append(args, "23")
	payload2, err2 := api.GeneratePayload(SimpleStorageABI, "set", args)
	assert.NoError(t, err2)
	assert.Equal(t, payload2, RemixSet23Payload)

	// Invalid method test
	payload3, err3 := api.GeneratePayload(SimpleStorageABI, "add", args)
	assert.Error(t, err3)
	assert.Empty(t, payload3)

	// Invalid parameter type test
	args1 := append(args, "123")
	payload4, err4 := api.GeneratePayload(SimpleStorageABI, "set", args1)
	assert.Error(t, err4)
	assert.Empty(t, payload4)

	// Invalid abiJSON string test
	payload5, err5 := api.GeneratePayload("SimpleStorageABI:asdf", "set", args)
	assert.Error(t, err5)
	assert.Empty(t, payload5)
}
