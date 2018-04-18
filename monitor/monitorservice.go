/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package monitor

import (
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele"
)

// MonitorService implement some rpc interfaces used by a monitor server
type MonitorService struct {
	p2pServer *p2p.Server         // Peer-to-Peer server infos
	seele     *seele.SeeleService // seele full node service
	seeleNode *node.Node          // seele node
	log       *log.SeeleLog

	rpcAddr string // listening port
	name    string // name display on the moitor
	node    string // node name
	version string // version
}

// New returns a rpc service
func New(seeleService *seele.SeeleService, seeleNode *node.Node, conf *node.Config, slog *log.SeeleLog, name string) (*MonitorService, error) {
	return &MonitorService{
		seele:     seeleService,
		seeleNode: seeleNode,
		log:       slog,
		name:      name,
		rpcAddr:   conf.RPCAddr,
		node:      conf.Name,
		version:   conf.Version,
	}, nil
}

// Protocols implements node.Service, nil as it dosn't use p2pservice
func (s *MonitorService) Protocols() []p2p.Protocol { return nil }

// Start implements node.Service, starting goroutines needed by SeeleService.
func (s *MonitorService) Start(srvr *p2p.Server) error {
	s.p2pServer = srvr

	s.log.Info("monitor rpc service start")

	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *MonitorService) Stop() error {

	return nil
}

// APIs implements node.Service, returning the collection of RPC services the seele package offers.
func (s *MonitorService) APIs() (apis []rpc.API) {
	return append(apis, []rpc.API{
		{
			Namespace: "monitor",
			Version:   "1.0",
			Service:   NewPublicMonitorAPI(s),
			Public:    true,
		},
	}...)
}
