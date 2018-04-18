/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package monitor

import (
	"runtime"
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
	var (
		mining  bool
		syncing bool
	)

	mining = api.s.seeleNode.Miner().IsMining()
	syncing = true

	result = &NodeStats{
		Active:   true,
		Syncing:  syncing,
		Mining:   mining,
		Hashrate: 20,
		Peers:    api.s.p2pServer.PeerCount(),
		Price:    10,
		Uptime:   100,
	}

	return nil
}
