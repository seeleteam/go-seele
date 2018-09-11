/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	rpc "github.com/seeleteam/go-seele/rpc2"
	"github.com/seeleteam/go-seele/seele"
)

// ServiceServer implements light server service.
type ServiceServer struct {
	p2pServer      *p2p.Server
	seeleProtocol  *LightProtocol
	log            *log.SeeleLog
	networkID      uint64
	chain          *core.Blockchain
	miner          *miner.Miner
	txPool         *core.TransactionPool
	accountStateDB database.Database // database used to store account state info.
	debtPool       *core.DebtPool
}

// NewServiceServer create ServiceServer
func NewServiceServer(service *seele.SeeleService, conf *node.Config, log *log.SeeleLog) (*ServiceServer, error) {
	seeleProtocol, err := NewLightProtocol(conf.P2PConfig.NetworkID, service.TxPool(), service.BlockChain(), true, nil, log)
	if err != nil {
		return nil, err
	}

	s := &ServiceServer{
		log:            log,
		seeleProtocol:  seeleProtocol,
		networkID:      service.NetVersion(),
		chain:          service.BlockChain(),
		miner:          service.Miner(),
		txPool:         service.TxPool(),
		accountStateDB: service.AccountStateDB(),
		debtPool:       service.DebtPool(),
	}

	return s, nil
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *ServiceServer) Protocols() (protos []p2p.Protocol) {
	return append(protos, s.seeleProtocol.Protocol)
}

// Start implements node.Service, starting goroutines needed by ServiceServer.
func (s *ServiceServer) Start(srvr *p2p.Server) error {
	s.p2pServer = srvr

	s.seeleProtocol.Start()
	s.seeleProtocol.blockLoop()
	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *ServiceServer) Stop() error {
	s.seeleProtocol.Stop()
	return nil
}

// APIs implements node.Service, returning the collection of RPC services the seele package offers.
func (s *ServiceServer) APIs() (apis []rpc.API) {
	return append(apis, []rpc.API{
		{
			Namespace: "seele",
			Version:   "1.0",
			Service:   NewPublicSeeleAPI(s),
			Public:    true,
		},
		{
			Namespace: "txpool",
			Version:   "1.0",
			Service:   NewTransactionPoolAPI(s),
			Public:    true,
		},
		{
			Namespace: "network",
			Version:   "1.0",
			Service:   NewPrivateNetworkAPI(s),
			Public:    false,
		},
		{
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s),
			Public:    false,
		},
	}...)
}

func (pm *LightProtocol) chainHeaderChanged(e event.Event) {
	newHeader := e.(common.Hash)
	if newHeader.IsEmpty() {
		return
	}

	pm.chainHeaderChangeChannel <- newHeader
}

func (pm *LightProtocol) blockLoop() {
	pm.wg.Add(1)
	defer pm.wg.Done()
	event.ChainHeaderChangedEventMananger.AddAsyncListener(pm.chainHeaderChanged)
needQuit:
	for {
		select {
		case newHeader := <-pm.chainHeaderChangeChannel:
			// todo
			pm.log.Debug("blockLoop head changed. %s", newHeader)
		case <-pm.quitCh:
			break needQuit
		}
	}

	event.ChainHeaderChangedEventMananger.RemoveListener(pm.chainHeaderChanged)
}

func (s *ServiceServer) NetVersion() uint64            { return s.networkID }
func (s *ServiceServer) Miner() *miner.Miner           { return s.miner }
func (s *ServiceServer) DebtPool() *core.DebtPool      { return s.debtPool }
func (s *ServiceServer) TxPool() *core.TransactionPool { return s.txPool }
