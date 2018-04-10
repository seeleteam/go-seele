/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"errors"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
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
	*addr = api.s.Coinbase
	return nil
}

type AddTxArgs struct {
	To     common.Address
	Amount uint64
}

// AddTx add a transaction to this node
func (api *PublicSeeleAPI) AddTx(args *AddTxArgs, result *bool) error {
	from, privateKey, err := crypto.GenerateKeyPair() // @todo actually we should use coinbase, but we could find coinbase's private key now

	var number big.Int
	number.SetUint64(args.Amount)
	tx := types.NewTransaction(*from, args.To, &number, 0) // @todo we also need to find the latest nonce
	tx.Sign(privateKey)

	err = api.s.txPool.AddTransaction(tx)
	if err != nil {
		*result = false
		return err
	}

	*result = true
	return nil
}

func (api *PublicSeeleAPI) GetBalance(args interface{}, result *big.Int) error {
	state := api.s.chain.CurrentState()
	amount, _ := state.GetAmount(api.s.Coinbase)
	result.Set(amount)
	return nil
}
