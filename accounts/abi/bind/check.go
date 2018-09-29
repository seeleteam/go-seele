/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package bind

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
)

// CheckInputArgs Check if the length and type of the input parameters match abi
func CheckInputArgs(abiArgs abi.Arguments, args ...interface{}) (bool, error) {
	if len(args) != len(abiArgs) {
		return false, fmt.Errorf("argument count mismatch: %d for %d", len(args), len(abiArgs))
	}

	for i, input := range abiArgs {
		if abiType, inputType := bindTypeGo(input.Type), reflect.TypeOf(args[i]).String(); abiType != inputType {
			return false, fmt.Errorf("index %d argument type mismatch: %s for %s", i, inputType, abiType)
		}
	}

	return true, nil
}

// ParseArgs parse the args string into the Seele type and return an error if
// the args length does not match or the parsed type fails.
func ParseArgs(abiArgs abi.Arguments, args []string) ([]interface{}, error) {
	if len(args) != len(abiArgs) {
		return nil, fmt.Errorf("argument count mismatch: %v(%d) for %v(%d)", args, len(args), abiArgs, len(abiArgs))
	}

	data := make([]interface{}, 0)
	for i, input := range abiArgs {
		arg, err := parseArg(bindTypeGo(input.Type), args[i])
		if err != nil {
			return nil, err
		}

		data = append(data, arg)
	}

	return data, nil
}

func parseArg(abiType string, arg string) (interface{}, error) {
	switch abiType {
	case "common.Address":
		bytes, err := hexutil.HexToBytes(arg)
		if err != nil {
			return nil, err
		}

		return common.BytesToAddress(bytes), nil
	case "*big.Int":
		number, ok := big.NewInt(0).SetString(arg, 10)
		if !ok {
			return nil, fmt.Errorf("number[%v] parsed error ", number)
		}

		return number, nil
	case "bool":
		if arg == "true" {
			return true, nil
		}

		return false, nil
	default:
		if strings.Contains(arg, "]byte") {
			return []byte(arg), nil
		}

		return arg, nil
	}
}
