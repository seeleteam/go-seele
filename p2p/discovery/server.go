/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"

	"github.com/seeleteam/go-seele/common"
)

func StartService(nodeDir string, myId common.Address, myAddr *net.UDPAddr, bootstrap []*Node, shard uint) *Database {
	udp := newUDP(myId, myAddr, shard)

	if bootstrap != nil {
		for _, bn := range bootstrap {
			udp.addNode(bn)
		}
		udp.trustNodes = bootstrap
	}
	udp.loadNodes(nodeDir)
	udp.StartServe(nodeDir)

	return udp.db
}
