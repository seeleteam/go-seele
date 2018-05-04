/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/p2p"
)

// PublicSeeleAPI provides an API to access full node-related information.
type PublicSeeleAPI struct {
	s *SeeleService
}

// NewPublicSeeleAPI creates a new PublicSeeleAPI object for rpc service.
func NewPublicSeeleAPI(s *SeeleService) *PublicSeeleAPI {
	return &PublicSeeleAPI{s}
}

// MinerInfo miner simple info
type MinerInfo struct {
	Coinbase           common.Address
	CurrentBlockHeight uint64
	HeaderHash         common.Hash
}

// GetInfo gets the account address that mining rewards will be send to.
func (api *PublicSeeleAPI) GetInfo(input interface{}, info *MinerInfo) error {
	block, _ := api.s.chain.CurrentBlock()

	*info = MinerInfo{
		Coinbase:           api.s.Coinbase,
		CurrentBlockHeight: block.Header.Height,
		HeaderHash:         block.HeaderHash,
	}

	return nil
}

// GetBalance get balance of the account. if the account's address is empty, will get the coinbase balance
func (api *PublicSeeleAPI) GetBalance(account *common.Address, result *big.Int) error {
	if account == nil || account.Equal(common.Address{}) {
		*account = api.s.Coinbase
	}

	state := api.s.chain.CurrentState()
	balance := state.GetBalance(*account)
	result.Set(balance)
	return nil
}

// AddTx add a tx to miner
func (api *PublicSeeleAPI) AddTx(tx *types.Transaction, result *bool) error {
	err := api.s.txPool.AddTransaction(tx)
	if err != nil {
		*result = false
		return err
	}

	*result = true
	return nil
}

// GetAccountNonce get account next used nonce
func (api *PublicSeeleAPI) GetAccountNonce(account *common.Address, nonce *uint64) error {
	state := api.s.chain.CurrentState()
	*nonce = state.GetNonce(*account)

	return nil
}

// PublicNetworkAPI provides an API to access network information.
type PublicNetworkAPI struct {
	p2pServer      *p2p.Server
	networkVersion uint64
}

// NewPublicNetworkAPI creates a new PublicNetworkAPI object for rpc service.
func NewPublicNetworkAPI(p2pServer *p2p.Server, networkVersion uint64) *PublicNetworkAPI {
	return &PublicNetworkAPI{p2pServer, networkVersion}
}

// GetPeerCount returns the count of peers
func (n *PublicNetworkAPI) GetPeerCount(input interface{}, result *int) error {
	*result = n.p2pServer.PeerCount()
	return nil
}

// GetNetworkVersion returns the network version
func (n *PublicNetworkAPI) GetNetworkVersion(input interface{}, result *uint64) error {
	*result = n.networkVersion
	return nil
}
