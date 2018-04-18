/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package monitor

import (
	"runtime"
)

// PublicMonitorAPI provides an API to monitor server
type PublicMonitorAPI struct {
	s *Service
}

// NewPublicMonitorAPI create new PublicMonitorAPI
func NewPublicMonitorAPI(s *Service) *PublicMonitorAPI {
	return &PublicMonitorAPI{s}
}

// NodeInfo return NodeInfo struct of the local node
func (api *PublicMonitorAPI) NodeInfo(arg int, result *NodeInfo) error {

	*result = NodeInfo{
		Name:     api.s.name,
		Node:     api.s.node,
		Port:     0, //api.s.p2pServer.ListenAddr,
		Network:  api.s.seele.NetVersion(),
		Protocol: "1.0",
		API:      "No",
		Os:       runtime.GOOS,
		OsVer:    runtime.GOARCH,
		Client:   api.s.version,
		History:  true,
	}

	return nil
}

// NodeStats return the information about the local node.
func (api *PublicMonitorAPI) NodeStats(arg int, result *NodeStats) error {
	var (
		mining   bool
		hashrate int
		syncing  bool
		gasprice int
	)

	mining = api.s.seeleNode.Miner().IsMining()
	hashrate = 10
	syncing = true
	gasprice = 20

	//mining = api.s.seele.Miner().IsMining()
	result = &NodeStats{
		Active:   true,
		Syncing:  syncing,
		Mining:   mining,
		Hashrate: hashrate,
		Peers:    api.s.p2pServer.PeerCount(),
		GasPrice: gasprice,
		Uptime:   0,
	}

	return nil
}
