/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"encoding/json"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

const (
	// CmdSubChainRegister register a sub-chain
	CmdSubChainRegister byte = iota
	// CmdSubChainQuery query a sub-chain.
	CmdSubChainQuery

	gasSubChainRegister = uint64(100000) // gas to register a sub-chain.
	gasSubChainQuery    = uint64(200000) // gas to query sub-chain information.
)

var (
	subChainCommands = map[byte]*cmdInfo{
		CmdSubChainRegister: &cmdInfo{gasSubChainRegister, registerSubChain},
		CmdSubChainQuery:    &cmdInfo{gasSubChainQuery, querySubChain},
	}
)

// SubChainInfo represents the sub-chain registration information.
type SubChainInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	StaticNodes []*discovery.Node `json:"staticNodes"`

	TokenFullName  string `json:"tokenFullName"`
	TokenShortName string `json:"tokenShortName"`
	TokenAmount    uint64 `json:"tokenAmount"`

	GenesisDifficulty uint64                      `json:"genesisDifficulty"`
	GenesisAccounts   map[common.Address]*big.Int `json:"genesisAccounts"`
	CreateTimestamp   *big.Int                    `json:"timestamp,omitempty"`

	// SubChain owner publick key
	Owner common.Address `json:"owner,omitempty"`
}

func registerSubChain(jsonRegInfo []byte, context *Context) ([]byte, error) {
	var info SubChainInfo
	if err := json.Unmarshal(jsonRegInfo, &info); err != nil {
		return nil, err
	}

	key, err := domainNameToKey([]byte(info.Name))
	if err != nil {
		return nil, err
	}

	if value := context.statedb.GetData(SubChainContractAddress, key); len(value) > 0 {
		return nil, errExists
	}

	// validate the reg info
	if len(info.Version) == 0 || len(info.TokenFullName) == 0 || len(info.TokenShortName) == 0 || info.TokenAmount == 0 {
		return nil, errInvalidSubChainInfo
	}

	// set transaction sender to subchain owner
	info.Owner = context.tx.Data.From
	info.CreateTimestamp = context.BlockHeader.CreateTimestamp

	value, err := json.MarshalIndent(info, "", "\t")
	if err != nil {
		return nil, err
	}

	context.statedb.CreateAccount(SubChainContractAddress)
	context.statedb.SetData(SubChainContractAddress, key, value)

	return nil, nil
}

func querySubChain(subChainName []byte, context *Context) ([]byte, error) {
	key, err := domainNameToKey(subChainName)
	if err != nil {
		return nil, err
	}

	return context.statedb.GetData(SubChainContractAddress, key), nil
}
