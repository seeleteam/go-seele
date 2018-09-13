/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package main

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/rpc2"
)

func registerSubChain(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"

	if err := system.ValidateDomainName([]byte(subChainNameVale)); err != nil {
		return nil, nil, err
	}

	if len(subChainVersionValue) == 0 {
		return nil, nil, errors.New("invalid subchain version")
	}

	if len(subChainTokenFullNameValue) == 0 {
		return nil, nil, errors.New("invalid subchain token full name")
	}

	if len(subChainTokenShortNameValue) == 0 {
		return nil, nil, errors.New("invalid subchain token short name")
	}

	if subChainTokenAmountValue == 0 {
		return nil, nil, errors.New("invalid subchain token amount")
	}

	mapAccounts := make(map[common.Address]*big.Int)
	for _, account := range subChainGenesisAccountsValue.Value() {
		arrayAccount := strings.Split(account, ":")
		if len(arrayAccount) != 2 {
			return nil, nil, errors.New("invalid genesis account")
		}

		addr, err := common.HexToAddress(arrayAccount[0])
		if err != nil {
			return nil, nil, errors.New("invalid genesis account address")
		}

		amount, ok := big.NewInt(0).SetString(arrayAccount[1], 10)
		if !ok {
			return nil, nil, errors.New("invalid genesis account amount")
		}

		mapAccounts[addr] = amount
	}

	subChain := system.SubChainInfo{
		Name:              subChainNameVale,
		Version:           subChainVersionValue,
		StaticNodes:       subChainStaticNodesValue.Value(),
		TokenFullName:     subChainTokenFullNameValue,
		TokenShortName:    subChainTokenShortNameValue,
		TokenAmount:       subChainTokenAmountValue,
		GenesisDifficulty: subChainGenesisDifficultyValue,
		GenesisAccounts:   mapAccounts,
	}

	subChainBytes, err := json.Marshal(&subChain)
	if err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.SubChainContractAddress, system.CmdSubChainRegister, subChainBytes)
	if err != nil {
		return nil, nil, err
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	output["SubChainName"] = subChain.Name
	output["TokenFullName"] = subChain.TokenFullName
	output["TokenShortName"] = subChain.TokenShortName

	return output, tx, err
}

func querySubChain(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"

	if err := system.ValidateDomainName([]byte(subChainNameVale)); err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.SubChainContractAddress, system.CmdSubChainQuery, []byte(subChainNameVale))
	if err != nil {
		return nil, nil, err
	}

	return tx, tx, err
}
