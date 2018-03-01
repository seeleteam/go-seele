/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import "github.com/seeleteam/go-seele/p2p"

// Service registers to node after node starts.
type Service interface {
	Protocols() []p2p.Protocol

	Start(server *p2p.Server) error

	Stop() error
}
