/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"sync"

	"github.com/seeleteam/go-seele/common"
)

func StartService(myId common.Address, myAddr *net.UDPAddr, bootstrap *Node) {
	udp := newUDP(myId, myAddr)

	if bootstrap != nil {
		udp.addNode(bootstrap)
	}

	udp.StartServe()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
