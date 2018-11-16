/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/seeleteam/go-seele/common/hexutil"

	"github.com/seeleteam/go-seele/common"
)

type solMethod struct {
	ShortName string
	FullName  string
	Hash      string
	ArgTypes  []string
}

func newSolMethod(funcFullName, funcHash string) solMethod {
	method := solMethod{
		ShortName: funcFullName,
		FullName:  funcFullName,
		Hash:      ensurePrefix(funcHash, "0x"),
		ArgTypes:  make([]string, 0),
	}

	idx := strings.IndexByte(funcFullName, '(')
	if idx < 0 {
		panic("the funcFullName does not contain the left bracket.")
	}

	if funcFullName[len(funcFullName)-1] != ')' {
		panic("the funcFullName does not end with right bracket.")
	}

	method.ShortName = funcFullName[:idx]
	methodArgs := funcFullName[idx+1 : len(funcFullName)-1]

	for _, t := range strings.Split(methodArgs, ",") {
		if t = strings.Trim(t, " "); len(t) > 0 {
			method.ArgTypes = append(method.ArgTypes, t)
		}
	}

	return method
}

func (m *solMethod) createInput() string {
	input := []string{m.Hash}
	methodArgs := os.Args[len(os.Args)-len(m.ArgTypes):]

	for i, t := range m.ArgTypes {
		encodedArg, err := m.encodeInput(t, methodArgs[i])
		if err != nil {
			fmt.Printf("Failed to parse %v argument, value = %v, error = %v\n", t, methodArgs[i], err.Error())
			return ""
		}

		input = append(input, encodedArg)
	}

	return strings.Join(input, "")
}

func (m *solMethod) encodeInput(argType, argValue string) (string, error) {
	switch {
	case argType == "bool":
		boolValue, err := strconv.ParseBool(argValue)
		if err != nil {
			return "", err
		}

		if boolValue {
			return m.encodeValue(big.NewInt(1)), nil
		}

		return m.encodeValue(big.NewInt(0)), nil
	case strings.HasPrefix(argType, "int") && argType[len(argType)-1] != ']':
		intValue, err := strconv.ParseInt(argValue, 10, 64)
		if err != nil {
			return "", err
		}

		return m.encodeValue(big.NewInt(intValue)), nil
	case strings.HasPrefix(argType, "uint") && argType[len(argType)-1] != ']':
		uintValue, err := strconv.ParseUint(argValue, 10, 64)
		if err != nil {
			return "", err
		}

		return m.encodeValue(new(big.Int).SetUint64(uintValue)), nil
	case argType == "address":
		addr, err := common.HexToAddress(argValue)
		if err != nil {
			return "", err
		}

		return m.encodeValue(addr.Big()), nil
	case strings.HasPrefix(argType, "byte"):
		b, err := hexutil.HexToBytes(argValue)
		if err != nil {
			return "", err
		}

		return m.encodeValue(new(big.Int).SetBytes(b)), nil
	default:
		return "", errors.New("not implemented yet")
	}
}

func (m *solMethod) encodeValue(num *big.Int) string {
	return common.BigToHash(num).Hex()[2:]
}
