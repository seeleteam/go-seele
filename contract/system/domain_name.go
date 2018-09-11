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
	gasRegisterDomainName  = uint64(50000)  // gas used to register a domain name
	gasDomainNameRegistrar = uint64(100000) // gas used to query the registrar of given domain name

	// CmdRegisterDomainName register a domain name
	CmdRegisterDomainName byte = iota
	// CmdDomainNameRegistrar query the registrar of specified domain name
	CmdDomainNameRegistrar
)

var (
	errNameEmpty   = errors.New("name is empty")
	errNameTooLong = errors.New("name too long")

	maxDomainNameLength = len(common.EmptyHash)

	domainNameCommands = map[byte]*cmdInfo{
		CmdRegisterDomainName:  &cmdInfo{gasRegisterDomainName, registerDomainName},
		CmdDomainNameRegistrar: &cmdInfo{gasDomainNameRegistrar, domainNameRegistrar},
	}
)

func registerDomainName(domainName []byte, context *Context) ([]byte, error) {
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
	err := ValidateDomainName(domainName)
	if err != nil {
		return common.EmptyHash, err
	}

	return common.BytesToHash(domainName), nil
}

func domainNameRegistrar(domainName []byte, context *Context) ([]byte, error) {
	key, err := domainNameToKey(domainName)
	if err != nil {
		return nil, err
	}

	return context.statedb.GetData(DomainNameContractAddress, key), nil
}

// ValidateDomainName validate domain name
func ValidateDomainName(domainName []byte) error {
	nameLen := len(domainName)

	if nameLen == 0 {
		return errNameEmpty
	}

	if nameLen > maxDomainNameLength {
		return errNameTooLong
	}

	return nil
}
