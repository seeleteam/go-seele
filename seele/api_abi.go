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
	"github.com/seeleteam/go-seele/common/hexutil"
)

// GeneratePayload according to abi json string and methodName and args to generate payload hex string
func (api *PublicSeeleAPI) GeneratePayload(abiJSON string, methodName string, args []string) (string, error) {
	parsed, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", fmt.Errorf("invalid abiJSON '%s', err: %s", abiJSON, err)
	}

	method, exist := parsed.Methods[methodName]
	if !exist {
		return "", fmt.Errorf("method '%s' not found", methodName)
	}

	seeleTypeArgs, err := bind.ParseArgs(method.Inputs, args)
	if err != nil {
		return "", err
	}

	bytes, err := parsed.Pack(methodName, seeleTypeArgs...)
	if err != nil {
		return "", err
	}

	return hexutil.BytesToHex(bytes), nil
}
