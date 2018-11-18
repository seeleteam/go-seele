/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

 
package pprof
// ProfService implements some rpc interfaces provided by a monitor server
type ProfService struct {
	// Peer-to-Peer server infos
	p2pServer *p2p.Server       
	// seele full node service  
	seele     *seele.SeeleService 
	// log
	log       *log.SeeleLog
	
}

Protocols() []p2p.Protocol

	APIs() (apis []rpc.API)

	Start(server *p2p.Server) error

	Stop() error
