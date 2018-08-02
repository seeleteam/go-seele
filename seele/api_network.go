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
func (n *PrivateNetworkAPI) GetPeerCount() (int, error) {
	return n.s.p2pServer.PeerCount(), nil
}

// GetNetworkVersion returns the network version
func (n *PrivateNetworkAPI) GetNetworkVersion() (uint64, error) {
	return n.s.NetVersion(), nil
}

// GetProtocolVersion returns the current seele protocol version this node supports
func (n *PrivateNetworkAPI) GetProtocolVersion() (uint, error) {
	return n.s.seeleProtocol.Protocol.Version, nil
}
