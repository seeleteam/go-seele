/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"fmt"
	"strings"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/accounts/abi/bind"
	"github.com/seeleteam/go-seele/common"
)

// KeyABIHash is the hash key to storing abi to statedb
var KeyABIHash = common.StringToHash("KeyABIHash")

// GeneratePayload according to abi to generate payload
func (api *PublicSeeleAPI) GeneratePayload(parsed abi.ABI, methodName string, args ...interface{}) ([]byte, error) {
	method, exist := parsed.Methods[methodName]
	if !exist {
		return nil, fmt.Errorf("method '%s' not found", methodName)
	}

	if ok, err := bind.CheckInputArgs(method.Inputs, args...); !ok {
		return nil, err
	}

	return parsed.Pack(methodName, args...)
}

// GetABI according to contract address to get abi
func (api *PublicSeeleAPI) GetABI(contractAddr common.Address) (abi.ABI, error) {
	parsed := abi.ABI{}
	statedb, err := api.s.chain.GetCurrentState()
	if err != nil {
		return parsed, err
	}

	abiBytes := statedb.GetData(contractAddr, KeyABIHash)
	if len(abiBytes) == 0 {
		return parsed, fmt.Errorf("the abi of contract '%s' not found", contractAddr.ToHex())
	}

	return abi.JSON(strings.NewReader(string(abiBytes)))
}
