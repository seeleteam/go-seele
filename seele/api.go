/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package seele

import (
	"github.com/seeleteam/go-seele/common"
)

// PublicSeeleAPI provides an API to access full node-related information.
type PublicSeeleAPI struct {
	s *SeeleService
}

// NewPublicSeeleAPI creates a new PublicSeeleAPI object for rpc service.
func NewPublicSeeleAPI(s *SeeleService) *PublicSeeleAPI {
	return &PublicSeeleAPI{s}
}

// Coinbase is the account address that mining rewards will be send to.
func (api *PublicSeeleAPI) Coinbase() (common.AccAddress, error) {
	return api.s.coinbase, nil
}
