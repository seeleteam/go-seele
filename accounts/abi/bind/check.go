/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package bind

import (
	"fmt"
	"reflect"

	"github.com/seeleteam/go-seele/accounts/abi"
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
