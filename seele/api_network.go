/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import "github.com/seeleteam/go-seele/p2p"

// PrivateNetworkAPI provides an API to access network information.
type PrivateNetworkAPI struct {
	s *SeeleService
}

// NewPrivateNetworkAPI creates a new PrivateNetworkAPI object for rpc service.
func NewPrivateNetworkAPI(s *SeeleService) *PrivateNetworkAPI {
	return &PrivateNetworkAPI{s}
}

// GetPeersInfo returns all the information of peers at the protocol granularity.
func (n *PrivateNetworkAPI) GetPeersInfo() ([]p2p.PeerInfo, error) {
	return n.s.p2pServer.PeersInfo(), nil
}

// GetPeerCount returns the count of peers
func (n *PrivateNetworkAPI) GetPeerCount(input interface{}, result *int) error {
	*result = n.s.p2pServer.PeerCount()
	return nil
}

// GetNetworkVersion returns the network version
func (n *PrivateNetworkAPI) GetNetworkVersion(input interface{}, result *uint64) error {
	*result = n.s.NetVersion()
	return nil
}

// GetProtocolVersion returns the current seele protocol version this node supports
func (n *PrivateNetworkAPI) GetProtocolVersion(input interface{}, result *uint) error {
	*result = n.s.seeleProtocol.Protocol.Version
	return nil
}
