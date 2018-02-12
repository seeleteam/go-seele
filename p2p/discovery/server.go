/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"fmt"
	"net"
	"sync"

	"github.com/seeleteam/go-seele/common"
)

// StartServerFat used by p2p.Server to start discovery service
func StartServerFat(port string, id string, nodeArr []*Node) (db *Database) {
	myId := common.HexToAddress(id)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%s", port))
	udp := newUDP(myId, addr)
	for _, node := range nodeArr {
		udp.addNode(node)
	}

	udp.StartServe()
	return udp.db
}

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
