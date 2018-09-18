/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/rpc"
)

var (
	errInvalidVersion        = errors.New("invalid subchain version")
	errInvalidTokenFullName  = errors.New("invalid subchain token full name")
	errInvalidTokenShortName = errors.New("invalid subchain token short name")
	errInvalidTokenAmount    = errors.New("invalid subchain token amount")
)

func registerSubChain(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"

	subChain, err := getSubChainFromFile(subChainJSONFileVale)
	if err != nil {
		return nil, nil, err
	}

	if err := system.ValidateDomainName([]byte(subChain.Name)); err != nil {
		return nil, nil, err
	}

	if len(subChain.Version) == 0 {
		return nil, nil, errInvalidVersion
	}

	if len(subChain.TokenFullName) == 0 {
		return nil, nil, errInvalidTokenFullName
	}

	if len(subChain.TokenShortName) == 0 {
		return nil, nil, errInvalidTokenShortName
	}

	if subChain.TokenAmount == 0 {
		return nil, nil, errInvalidTokenAmount
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

	if err := system.ValidateDomainName([]byte(nameValue)); err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.SubChainContractAddress, system.CmdSubChainQuery, []byte(nameValue))
	if err != nil {
		return nil, nil, err
	}

	return tx, tx, err
}

func getSubChainFromFile(filepath string) (*system.SubChainInfo, error) {
	var subChain system.SubChainInfo
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return &subChain, err
	}

	err = json.Unmarshal(buff, &subChain)
	return &subChain, err
}
