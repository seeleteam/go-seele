/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"fmt"
	"strings"

	"github.com/seeleteam/go-seele/common/hexutil"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/accounts/abi/bind"
	"github.com/seeleteam/go-seele/common"
)

// KeyABIHash is the hash key to storing abi to statedb
var KeyABIHash = common.StringToHash("KeyABIHash")

// GeneratePayload according to abi to generate payload hex string
func (api *PublicSeeleAPI) GeneratePayload(abiJSON string, methodName string, args ...interface{}) (string, error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", fmt.Errorf("invalid abiJSON '%s', err: %s", abiJSON, err)
	}

	method, exist := parsed.Methods[methodName]
	if !exist {
		return "", fmt.Errorf("method '%s' not found", methodName)
	}

	if ok, err := bind.CheckInputArgs(method.Inputs, args...); !ok {
		return "", err
	}

	bytes, err := parsed.Pack(methodName, args...)
	if err != nil {
		return "", err
	}

	return hexutil.BytesToHex(bytes), nil
}

// GetABI according to contract address to get abi json string
func (api *PublicSeeleAPI) GetABI(contractAddr common.Address) (string, error) {
	statedb, err := api.s.chain.GetCurrentState()
	if err != nil {
		return "", err
	}

	abiBytes := statedb.GetData(contractAddr, KeyABIHash)
	if len(abiBytes) == 0 {
		return "", fmt.Errorf("the abi of contract '%s' not found", contractAddr.ToHex())
	}

	return string(abiBytes), nil
}
