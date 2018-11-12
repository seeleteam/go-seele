/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package api

import "github.com/seeleteam/go-seele/p2p"

// PrivateNetworkAPI provides an API to access network information.
type PrivateNetworkAPI struct {
	s Backend
}

// NewPrivateNetworkAPI creates a new PrivateNetworkAPI object for rpc service.
func NewPrivateNetworkAPI(s Backend) *PrivateNetworkAPI {
	return &PrivateNetworkAPI{s}
}

// GetPeersInfo returns all the information of peers at the protocol granularity.
func (n *PrivateNetworkAPI) GetPeersInfo() ([]p2p.PeerInfo, error) {
	return n.s.GetP2pServer().PeersInfo(), nil
}

// GetPeerCount returns the count of peers
func (n *PrivateNetworkAPI) GetPeerCount() (int, error) {
	return n.s.GetP2pServer().PeerCount(), nil
}

// GetNetVersion returns the net version
func (n *PrivateNetworkAPI) GetNetVersion() (string, error) {
	return n.s.GetNetVersion(), nil
}

// GetNetworkID returns the network ID, unique mark of seele Network
func (n *PrivateNetworkAPI) GetNetworkID() (string, error) {
	return n.s.GetNetWorkID(), nil
}

// GetProtocolVersion returns the current seele protocol version this node supports
func (n *PrivateNetworkAPI) GetProtocolVersion() (uint, error) {
	return n.s.ProtocolBackend().GetProtocolVersion()
}
