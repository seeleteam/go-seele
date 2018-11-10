/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package bind

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
)

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
	case "int8":
		number, err := strconv.Atoi(arg)
		if err != nil {
			return nil, err
		}
		return int8(number), nil
	case "int16":
		number, err := strconv.Atoi(arg)
		if err != nil {
			return nil, err
		}
		return int16(number), nil
	case "int32":
		number, err := strconv.Atoi(arg)
		if err != nil {
			return nil, err
		}
		return int32(number), nil
	case "int64":
		number, err := strconv.Atoi(arg)
		if err != nil {
			return nil, err
		}
		return int64(number), nil
	case "uint8":
		number, err := strconv.Atoi(arg)
		if err != nil {
			return nil, err
		}
		return uint8(number), nil
	case "uint16":
		number, err := strconv.Atoi(arg)
		if err != nil {
			return nil, err
		}
		return uint16(number), nil
	case "uint32":
		number, err := strconv.Atoi(arg)
		if err != nil {
			return nil, err
		}
		return uint32(number), nil
	case "uint64":
		number, err := strconv.Atoi(arg)
		if err != nil {
			return nil, err
		}
		return uint64(number), nil
	default:
		if strings.Contains(abiType, "]byte") {
			fmt.Println("abiType:", abiType)
			fmt.Println("arg:", arg)
			bytes, err := hexutil.HexToBytes(arg)
			if err != nil {
				return nil, err
			}
			length := abiType[1 : len(abiType)-5]
			if length == "" {
				return bytes, nil
			}
			// b := [l]byte{}
			switch length {
			case "32":
				var b [32]byte
				for index, count := len(bytes)-1, len(b)-1; index >= 0 && count >= 0; {
					b[count] = bytes[index]
					index--
					count--
				}
				return b, nil
			default:
				return nil, errors.New("Now abi only supports bytes32 and bytes, and it will totally support in seele.js")
			}
		}

		return arg, nil
	}
}
