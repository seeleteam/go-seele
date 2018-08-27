/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import "errors"

const (
	gasCreateDomainName = uint64(500000)

	maxDomainNameLength = 512
)

var (
	errDomainNameEmpty   = errors.New("domain name is empty")
	errDomainNameTooLong = errors.New("domain name length exceed 512")
)

type domainNameContract struct{}

func (contract *domainNameContract) Run(input []byte, context Context) ([]byte, uint64, error) {
	result, err := createDomainName(input, context)
	return result, gasCreateDomainName, err
}

func createDomainName(domainName []byte, context Context) ([]byte, error) {
	if len(domainName) == 0 {
		return nil, errDomainNameEmpty
	}

	if len(domainName) > maxDomainNameLength {
		return nil, errDomainNameTooLong
	}

	return nil, nil
}
