/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"errors"

	"github.com/seeleteam/go-seele/common"
)

const (
	gasCreateDomainName  = uint64(50000)  // gas used to create a domain name
	gasDomainNameCreator = uint64(100000) // gas used to query the creator of given domain name

	cmdCreateDomainName  = byte(0) // create a domain name
	cmdDomainNameCreator = byte(1) // query the creator of specified domain name
)

var (
	errNameEmpty   = errors.New("name is empty")
	errNameTooLong = errors.New("name too long")

	maxDomainNameLength = len(common.EmptyHash)

	domainNameCommands = map[byte]*cmdInfo{
		cmdCreateDomainName:  &cmdInfo{gasCreateDomainName, createDomainName},
		cmdDomainNameCreator: &cmdInfo{gasDomainNameCreator, domainNameCreator},
	}
)

func createDomainName(domainName []byte, context *Context) ([]byte, error) {
	key, err := domainNameToKey(domainName)
	if err != nil {
		return nil, err
	}

	// create account in statedb for the first time.
	context.statedb.CreateAccount(DomainNameContractAddress)

	// ensure not exist
	if value := context.statedb.GetData(DomainNameContractAddress, key); len(value) > 0 {
		return nil, errExists
	}

	// save in statedb
	value := context.tx.Data.From.Bytes()
	context.statedb.SetData(DomainNameContractAddress, key, value)

	return nil, nil
}

func domainNameToKey(domainName []byte) (common.Hash, error) {
	nameLen := len(domainName)

	if nameLen == 0 {
		return common.EmptyHash, errNameEmpty
	}

	if nameLen > maxDomainNameLength {
		return common.EmptyHash, errNameTooLong
	}

	return common.BytesToHash(domainName), nil
}

func domainNameCreator(domainName []byte, context *Context) ([]byte, error) {
	key, err := domainNameToKey(domainName)
	if err != nil {
		return nil, err
	}

	return context.statedb.GetData(DomainNameContractAddress, key), nil
}
