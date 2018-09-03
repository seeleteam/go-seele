/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	rpc "github.com/seeleteam/go-seele/rpc2"
	"github.com/seeleteam/go-seele/seele"
)

// ServiceServer implements full node service.
type ServiceServer struct {
	p2pServer     *p2p.Server
	seeleProtocol *LightProtocol
	log           *log.SeeleLog
}

// NewServiceServer create ServiceServer
func NewServiceServer(service *seele.SeeleService, conf *node.Config, log *log.SeeleLog) (*ServiceServer, error) {
	seeleProtocol, err := NewLightProtocol(conf.P2PConfig.NetworkID, service.TxPool(), service.BlockChain(), true, log)
	if err != nil {
		return nil, err
	}

	s := &ServiceServer{
		log:           log,
		seeleProtocol: seeleProtocol,
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
	//todo
	return
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
			//
			pm.log.Debug("blockLoop head changed. %s", newHeader)
		case <-pm.quitCh:
			break needQuit
		}
	}
}
