/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"net"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
)

const (
	PINGPONGINTERVAL  = 10 * time.Second // sleep between ping pong
	DISCOVERYINTERVAL = 10 * time.Second // sleep between discovery
)

func StartServer(port, id string) {
	udp := getUDP(port, HexToNodeID(id))
	log.Debug("nodeid: %s", common.BytesToHex(udp.self.ID.Bytes()))

	udp.StartServe()

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func getUDP(port string, id NodeID) *udp {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		log.Fatal(err.Error())
	}

	return NewUDP(id, addr)
}

func HexToNodeID(id string) NodeID {
	byte, err := common.HexToBytes(id)
	if err != nil {
		log.Fatal(err.Error())
	}

	nid, err := BytesToID(byte)
	if err != nil {
		log.Fatal(err.Error())
	}

	return nid
}

func SendPing(port, id, targePort string) {
	myid := getRandomNodeID()

	udp := getUDP(port, myid)

	log.Debug("nodeid: %s", common.BytesToHex(udp.self.ID.Bytes()))

	addr := getAddr(targePort)
	tid := HexToNodeID(id)

	n := NewNodeWithAddr(tid, addr)
	udp.table.addNode(n)
	udp.db.add(n.sha, n)

	udp.StartServe()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
