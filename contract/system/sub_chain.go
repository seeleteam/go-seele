/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"encoding/json"

	"github.com/seeleteam/go-seele/common"
)

const (
	cmdSubChainRegister byte = iota // register a sub-chain.
	cmdSubChainQuery

	gasSubChainRegister = uint64(100000) // gas to register a sub-chain.
	gasSubChainQuery    = uint64(200000) // gas to query sub-chain information.
)

var (
	subChainCommands = map[byte]*cmdInfo{
		cmdSubChainRegister: &cmdInfo{gasSubChainRegister, registerSubChain},
		cmdSubChainQuery:    &cmdInfo{gasSubChainQuery, querySubChain},
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
	GenesisAccounts   map[common.Address]uint64
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

	if value := context.statedb.GetData(subChainContractAddress, key); len(value) > 0 {
		return nil, errExists
	}

	value, err := json.MarshalIndent(info, "", "\t")
	if err != nil {
		return nil, err
	}

	context.statedb.SetData(subChainContractAddress, key, value)

	return nil, nil
}

func querySubChain(subChainName []byte, context *Context) ([]byte, error) {
	key, err := domainNameToKey(subChainName)
	if err != nil {
		return nil, err
	}

	return context.statedb.GetData(subChainContractAddress, key), nil
}
