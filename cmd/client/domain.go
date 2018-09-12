/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/rpc2"
)

// createDomainName create a domain name
func createDomainName(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"
	if err := system.ValidateDomainName([]byte(domainNameValue)); err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.DomainNameContractAddress, system.CmdCreateDomainName, []byte(domainNameValue))
	if err != nil {
		return nil, nil, err
	}

	return tx, tx, err
}

// getDomainNameOwner get domain name owner
func getDomainNameOwner(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"

	if err := system.ValidateDomainName([]byte(domainNameValue)); err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.DomainNameContractAddress, system.CmdGetDomainNameOwner, []byte(domainNameValue))
	if err != nil {
		return nil, nil, err
	}

	return tx, tx, err
}
