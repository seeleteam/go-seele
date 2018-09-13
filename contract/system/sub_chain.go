/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"encoding/json"
	"math/big"

	"github.com/seeleteam/go-seele/common"
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
	Name        string
	Version     string
	StaticNodes []string

	TokenFullName  string
	TokenShortName string
	TokenAmount    uint64

	GenesisDifficulty uint64
	GenesisAccounts   map[common.Address]*big.Int
}

func registerSubChain(jsonRegInfo []byte, context *Context) ([]byte, error) {
	var info SubChainInfo
	if err := json.Unmarshal(jsonRegInfo, &info); err != nil {
		return nil, err
	}

	// @todo validate the reg info

	key, err := domainNameToKey([]byte(info.Name))
	if err != nil {
		return nil, err
	}

	if value := context.statedb.GetData(SubChainContractAddress, key); len(value) > 0 {
		return nil, errExists
	}

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
