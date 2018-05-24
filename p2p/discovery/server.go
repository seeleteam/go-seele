/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"

	"github.com/seeleteam/go-seele/common"
)

func StartService(myId common.Address, myAddr *net.UDPAddr, bootstrap map[string]*Node, shard uint) *Database {
	udp := newUDP(myId, myAddr, shard)

	if bootstrap != nil {
		for _, bn := range bootstrap {
			udp.addNode(bn)
		}
	}

	udp.StartServe()

	return udp.db
}
