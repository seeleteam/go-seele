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
)

var (
	domainNameContractAddress = common.BytesToAddress([]byte{1, 1})

	contracts = map[common.Address]Contract{
		domainNameContractAddress: &domainNameContract{},
	}
)
