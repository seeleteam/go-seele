/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/rpc"
)

// createDomainName create a domain name
func createDomainName(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"
	if err := system.ValidateDomainName([]byte(nameValue)); err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.DomainNameContractAddress, system.CmdCreateDomainName, []byte(nameValue))
	if err != nil {
		return nil, nil, err
	}

	return tx, tx, err
}

// getDomainNameOwner get domain name owner
func getDomainNameOwner(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"

	if err := system.ValidateDomainName([]byte(nameValue)); err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.DomainNameContractAddress, system.CmdGetDomainNameOwner, []byte(nameValue))
	if err != nil {
		return nil, nil, err
	}

	return tx, tx, err
}
