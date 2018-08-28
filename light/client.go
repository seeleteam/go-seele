/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"context"

	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	rpc "github.com/seeleteam/go-seele/rpc2"
)

// ServiceClient implements full node service.
type ServiceClient struct {
	networkID     uint64
	p2pServer     *p2p.Server
	seeleProtocol *LightProtocol
	log           *log.SeeleLog

	txPool         TransactionPool
	chain          BlockChain
	chainDB        database.Database // database used to store blocks.
	accountStateDB database.Database // database used to store account state info.
}

// ServiceContext is a collection of service configuration inherited from node
type ServiceContext struct {
	DataDir string
}

// NewServiceClient create ServiceClient
func NewServiceClient(ctx context.Context, conf *node.Config, log *log.SeeleLog) (s *ServiceClient, err error) {
	s = &ServiceClient{
		log:       log,
		networkID: conf.P2PConfig.NetworkID,
	}

	return s, nil
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *ServiceClient) Protocols() (protos []p2p.Protocol) {
	protos = append(protos, s.seeleProtocol.Protocol)
	return
}

// Start implements node.Service, starting goroutines needed by ServiceClient.
func (s *ServiceClient) Start(srvr *p2p.Server) error {
	s.p2pServer = srvr

	s.seeleProtocol.Start()
	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *ServiceClient) Stop() error {
	s.seeleProtocol.Stop()

	s.chainDB.Close()
	s.accountStateDB.Close()
	return nil
}

// APIs implements node.Service, returning the collection of RPC services the seele package offers.
func (s *ServiceClient) APIs() (apis []rpc.API) {
	return
}
