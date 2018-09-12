/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/rpc2"
)

func registerDomainName(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"

	if err := system.ValidateDomainName([]byte(domainNameValue)); err != nil {
		return nil, nil, err
	}

	return sendSystemContractTx(client, system.DomainNameContractAddress, system.CmdRegisterDomainName, []byte(domainNameValue))
}

func domainNameRegister(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"

	if err := system.ValidateDomainName([]byte(domainNameValue)); err != nil {
		return nil, nil, err
	}

	return sendSystemContractTx(client, system.DomainNameContractAddress, system.CmdDomainNameRegistrar, []byte(domainNameValue))
}

// sendSystemContractTx send system contract transaction
func sendSystemContractTx(client *rpc.Client, to common.Address, method byte, payload []byte) (map[string]interface{}, *types.Transaction, error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return nil, nil, err
	}

	txd.To = to
	txd.Payload = append([]byte{method}, payload...)
	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, nil, err
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	return output, tx, err
}
