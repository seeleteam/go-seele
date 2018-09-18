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
	tx          *types.Transaction
	statedb     *state.Statedb
	BlockHeader *types.BlockHeader
}

// NewContext creates a system contract context.
func NewContext(tx *types.Transaction, statedb *state.Statedb, BlockHeader *types.BlockHeader) *Context {
	return &Context{tx, statedb, BlockHeader}
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
	errInvalidCommand      = errors.New("invalid command")
	errExists              = errors.New("already exists")
	errInvalidSubChainInfo = errors.New("invalid SubChainInfo")

	// DomainNameContractAddress domain contract address
	DomainNameContractAddress = common.BytesToAddress([]byte{1, 1})
	// SubChainContractAddress subchain contract address
	SubChainContractAddress = common.BytesToAddress([]byte{1, 2})
	// HashTimeLockContractAddress HTLC contract address
	HashTimeLockContractAddress = common.BytesToAddress([]byte{1, 3})
	// MasternodeContractAddress masternode contract address
	MasternodeContractAddress = common.BytesToAddress([]byte{1, 4})
	// BTCRelayContractAddress btc-relay contract address
	BTCRelayContractAddress = common.BytesToAddress([]byte{1, 5})

	// Contracts are system contracts
	contracts = map[common.Address]Contract{
		DomainNameContractAddress:   &contract{domainNameCommands},
		SubChainContractAddress:     &contract{subChainCommands},
		HashTimeLockContractAddress: &contract{htlcCommands},
		MasternodeContractAddress:   &contract{masternodeCommands},
		BTCRelayContractAddress:     &contract{brCommands},
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

// GetContractByAddress get system contract by the address
func GetContractByAddress(address common.Address) Contract {
	return contracts[address]
}
