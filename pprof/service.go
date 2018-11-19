/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pprof

import (
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
)

// ProfService implements some rpc interfaces provided by a ProfService server
type ProfService struct {
	// log
	log *log.SeeleLog
}

// NewService returns a NewService instance
func NewService(slog *log.SeeleLog) (*ProfService, error) {
	return &ProfService{
		log: slog,
	}, nil
}

// Protocols return protocols
func (p *ProfService) Protocols() []p2p.Protocol {
	return nil
}

// APIs api of pprof http server
func (p *ProfService) APIs() (apis []rpc.API) {
	return append(apis, []rpc.API{
		{
			Namespace: "pprof",
			Version:   "1.0",
			Service:   NewProfServer(),
			Public:    false,
		},
	}...)
}

// Start start service
func (p *ProfService) Start(server *p2p.Server) error {
	p.log.Info("ProfService start...")

	return nil
}

// Stop stop service
func (p *ProfService) Stop() error {
	return nil
}
