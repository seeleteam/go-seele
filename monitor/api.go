/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package monitor

import (
	"errors"
	"runtime"
)

// error infos
var (
	ErrBlockchainInfoFailed = errors.New("get blockchain info failed")
	ErrMinerInfoFailed      = errors.New("get miner info failed")
	ErrNodeInfoFailed       = errors.New("get node info failed")
	ErrP2PServerInfoFailed  = errors.New("get p2p server info failed")
)

// PublicMonitorAPI provides an API to monitor service
type PublicMonitorAPI struct {
	s *MonitorService
}

// NewPublicMonitorAPI create new PublicMonitorAPI
func NewPublicMonitorAPI(s *MonitorService) *PublicMonitorAPI {
	return &PublicMonitorAPI{s}
}

// NodeInfo return NodeInfo struct of the local node
func (api *PublicMonitorAPI) NodeInfo(arg int, result *NodeInfo) error {
	*result = NodeInfo{
		Name:       api.s.name,
		Node:       api.s.node,
		Port:       0, //api.s.p2pServer.ListenAddr,
		NetVersion: api.s.seele.NetVersion(),
		Protocol:   "1.0",
		API:        "No",
		Os:         runtime.GOOS,
		OsVer:      runtime.GOARCH,
		Client:     api.s.version,
		History:    true,
	}

	return nil
}

// NodeStats return the information about the local node.
func (api *PublicMonitorAPI) NodeStats(arg int, result *NodeStats) error {
	if api.s.p2pServer == nil {
		return ErrP2PServerInfoFailed
	}

	if api.s.seeleNode == nil {
		return ErrNodeInfoFailed
	}

	if api.s.seele.Miner() == nil {
		return ErrMinerInfoFailed
	}

	mining := api.s.seele.Miner().IsMining()

	*result = NodeStats{
		Active:  true,
		Syncing: true,
		Mining:  mining,
		Peers:   api.s.p2pServer.PeerCount(),
	}

	return nil
}
