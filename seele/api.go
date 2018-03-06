/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package seele

import (
	"errors"

	"github.com/seeleteam/go-seele/common"
)

var (
	errAPIInvalidParams = errors.New("invalid api parameters")
)

// PublicSeeleAPI provides an API to access full node-related information.
type PublicSeeleAPI struct {
	s *SeeleService
}

// NewPublicSeeleAPI creates a new PublicSeeleAPI object for rpc service.
func NewPublicSeeleAPI(s *SeeleService) *PublicSeeleAPI {
	return &PublicSeeleAPI{s}
}

// Coinbase gets the account address that mining rewards will be send to.
func (api *PublicSeeleAPI) Coinbase(input interface{}, addr *common.Address) error {
	if addr == nil {
		return errAPIInvalidParams
	}
	*addr = api.s.coinbase
	return nil
}
