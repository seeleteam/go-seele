/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package bind

import (
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
			return nil, fmt.Errorf("number[%v] parsed error", arg)
		}
		return number, nil
	case "bool":
		if arg == "true" {
			return true, nil
		}
		return false, nil
	case "int8":
		number, err := strconv.ParseInt(arg, 10, 8)
		if err != nil {
			return nil, err
		}
		return int8(number), nil
	case "int16":
		number, err := strconv.ParseInt(arg, 10, 16)
		if err != nil {
			return nil, err
		}
		return int16(number), nil
	case "int32":
		number, err := strconv.ParseInt(arg, 10, 32)
		if err != nil {
			return nil, err
		}
		return int32(number), nil
	case "int64":
		number, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return nil, err
		}
		return int64(number), nil
	case "uint8":
		number, err := strconv.ParseUint(arg, 10, 8)
		if err != nil {
			return nil, err
		}
		return uint8(number), nil
	case "uint16":
		number, err := strconv.ParseUint(arg, 10, 16)
		if err != nil {
			return nil, err
		}
		return uint16(number), nil
	case "uint32":
		number, err := strconv.ParseUint(arg, 10, 32)
		if err != nil {
			return nil, err
		}
		return uint32(number), nil
	case "uint64":
		number, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return nil, err
		}
		return uint64(number), nil
	default:
		if strings.Contains(abiType, "]byte") {
			bytes, err := hexutil.HexToBytes(arg)
			if err != nil {
				return nil, err
			}
			length := abiType[1 : len(abiType)-5]
			if length == "" {
				return bytes, nil
			}
			switch length {
			case "32":
				l := len(bytes)
				if l > 32 {
					return nil, fmt.Errorf("bytes32[%s, length:%d] transfer overflow", arg, l)
				}
				var b [32]byte
				for index, count := l-1, len(b)-1; index >= 0 && count >= 0; {
					b[count] = bytes[index]
					index--
					count--
				}
				return b, nil
			default:
				return nil, fmt.Errorf("Now abi only supports bytes32 and bytes, and it will totally support in seele.js, reject: %s", length)
			}
		}

		return arg, nil
	}
}
