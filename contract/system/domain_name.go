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
	errDomainNameEmpty   = errors.New("domain name is empty")
	errDomainNameTooLong = errors.New("domain name too long")
	errDomainNameExists  = errors.New("domain name already exists")

	maxDomainNameLength = len(common.EmptyHash)
)

type domainNameContract struct{}

func (contract *domainNameContract) RequiredGas(input []byte) uint64 {
	if len(input) == 0 {
		return gasInvalidCommand
	}

	switch input[0] {
	case cmdCreateDomainName:
		return gasCreateDomainName
	case cmdDomainNameCreator:
		return gasDomainNameCreator
	default:
		return gasInvalidCommand
	}
}

func (contract *domainNameContract) Run(input []byte, context *Context) ([]byte, error) {
	if len(input) == 0 {
		return nil, errInvalidCommand
	}

	switch input[0] {
	case cmdCreateDomainName:
		return createDomainName(input[1:], context)
	case cmdDomainNameCreator:
		return domainNameCreator(input[1:], context)
	default:
		return nil, errInvalidCommand
	}
}

func createDomainName(domainName []byte, context *Context) ([]byte, error) {
	key, err := domainNameToKey(domainName)
	if err != nil {
		return nil, err
	}

	// create account in statedb for the first time.
	context.statedb.CreateAccount(domainNameContractAddress)

	// ensure not exist
	if value := context.statedb.GetData(domainNameContractAddress, key); len(value) > 0 {
		return nil, errDomainNameExists
	}

	// save in statedb
	value := context.tx.Data.From.Bytes()
	context.statedb.SetData(domainNameContractAddress, key, value)

	return nil, nil
}

func domainNameToKey(domainName []byte) (common.Hash, error) {
	nameLen := len(domainName)

	if nameLen == 0 {
		return common.EmptyHash, errDomainNameEmpty
	}

	if nameLen > maxDomainNameLength {
		return common.EmptyHash, errDomainNameTooLong
	}

	return common.BytesToHash(domainName), nil
}

func domainNameCreator(domainName []byte, context *Context) ([]byte, error) {
	key, err := domainNameToKey(domainName)
	if err != nil {
		return nil, err
	}

	return context.statedb.GetData(domainNameContractAddress, key), nil
}
