/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"net"
)

// connection TODO add band meter for connection
type connection struct {
	fd net.Conn // tcp connection
	//node *discovery.Node // remote peer that this peer connects
}
