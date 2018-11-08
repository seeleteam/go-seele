/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"errors"
	"regexp"
	"strings"

	"github.com/seeleteam/go-seele/common"
)

const (
	// CmdCreateDomainName create a domain
	CmdCreateDomainName byte = iota
	// CmdGetDomainNameOwner query the registrar of specified domain name
	CmdGetDomainNameOwner
)

const (
	// gas used to create a domain name
	gasCreateDomainName = uint64(50000)
	// gas used to get the owner of given domain
	gasGetDomainNameOwner = uint64(100000)
)

var (
	errNameEmpty   = errors.New("name is empty")
	errNameTooLong = errors.New("name too long")
	errInvalidName = errors.New("invalid name, only numbers, letters, and dash lines are allowed")

	maxDomainNameLength = len(common.EmptyHash)

	domainNameCommands = map[byte]*cmdInfo{
		CmdCreateDomainName:   &cmdInfo{gasCreateDomainName, createDomainName},
		CmdGetDomainNameOwner: &cmdInfo{gasGetDomainNameOwner, getDomainNameOwner},
	}
)

// createDomainName create a domain name
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

	return value, nil
}

// getDomainNameOwner get domain name owner
func getDomainNameOwner(domainName []byte, context *Context) ([]byte, error) {
	key, err := domainNameToKey(domainName)
	if err != nil {
		return nil, err
	}

	owner := context.statedb.GetData(DomainNameContractAddress, key)
	if len(owner) == 0 {
		return nil, errNotFound
	}

	return owner, nil
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

	ok, err := regexp.Match(`^[a-zA-Z0-9\-]+$`, domainName)
	if err != nil {
		return err
	}
	if !ok {
		return errInvalidName
	}
	return nil
}

// domainNameToKey convert domain name to hash and uppercase to lowercase
func domainNameToKey(domainName []byte) (common.Hash, error) {
	lowerDomainName := []byte(strings.ToLower(string(domainName)))

	err := ValidateDomainName(lowerDomainName)
	if err != nil {
		return common.EmptyHash, err
	}

	return common.BytesToHash(domainName), nil
}
