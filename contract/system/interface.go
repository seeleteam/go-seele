/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"errors"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
)

// Context provides information that required in system contract.
type Context struct {
	tx      *types.Transaction
	statedb *state.Statedb
}

// NewContext creates a system contract context.
func NewContext(tx *types.Transaction, statedb *state.Statedb) *Context {
	return &Context{tx, statedb}
}

// Contract is the basic interface for native Go contracts in Seele.
type Contract interface {
	RequiredGas(input []byte) uint64
	Run(input []byte, context *Context) ([]byte, error)
}

const (
	gasInvalidCommand = uint64(50000)
)

var (
	errInvalidCommand = errors.New("invalid command")

	domainNameContractAddress = common.BytesToAddress([]byte{1, 1})
	subChainContractAddress   = common.BytesToAddress([]byte{1, 2})

	contracts = map[common.Address]Contract{
		domainNameContractAddress: &contract{domainNameCommands},
		subChainContractAddress:   &contract{subChainCommands},
	}
)

type handler func([]byte, *Context) ([]byte, error)

type cmdInfo struct {
	cmdUsedGas uint64
	cmdHandler handler
}

type contract struct {
	cmds map[byte]*cmdInfo
}

func (c *contract) RequiredGas(input []byte) uint64 {
	if len(input) == 0 {
		return gasInvalidCommand
	}

	if info, found := c.cmds[input[0]]; found {
		return info.cmdUsedGas
	}

	return gasInvalidCommand
}

func (c *contract) Run(input []byte, context *Context) ([]byte, error) {
	if len(input) == 0 {
		return nil, errInvalidCommand
	}

	if info, found := c.cmds[input[0]]; found {
		return info.cmdHandler(input[1:], context)
	}

	return nil, errInvalidCommand
}
