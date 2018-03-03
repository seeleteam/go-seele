/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
)

// SeeleService implements full node service.
type SeeleService struct {
	networkID     uint64
	seeleProtocol *SeeleProtocol
	log           *log.SeeleLog
}

// NewSeeleService create SeeleService
func NewSeeleService(networkID uint64, log *log.SeeleLog) (s *SeeleService, err error) {
	s = &SeeleService{
		networkID: networkID,
		log:       log,
	}

	s.seeleProtocol, err = NewSeeleProtocol(networkID, log)
	return s, err
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *SeeleService) Protocols() (protos []p2p.ProtocolInterface) {
	protos = append(protos, s.seeleProtocol)
	return
}

// Start implements node.Service, starting goroutines needed by SeeleService.
func (s *SeeleService) Start(srvr *p2p.Server) error {
	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *SeeleService) Stop() error {
	s.seeleProtocol.Stop()
	return nil
}

// APIs implements node.Service, returning the collection of RPC services the seele package offers.
func (s *SeeleService) APIs() (apis []rpc.API) {
	return apis
}
