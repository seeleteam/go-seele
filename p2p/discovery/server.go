/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"sync"

	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/log"
)

func StartService(port, id string) {
	var nodeid NodeID
	if id == "" {
		nodeid = getRandomNodeID()
	} else {
		nodeid = hexToNodeID(id)
	}

	udp := getUDP(port, nodeid)
	log.Debug("nodeid: %s", hexutil.BytesToHex(udp.self.ID.Bytes()))

	udp.StartServe()

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()

}

func StartServerFat(port, id string, nodeArr []*Node) (db *Database) {
	udp := getUDP(port, hexToNodeID(id))
	for _, node := range nodeArr {
		udp.table.addNode(node)
	}

	udp.StartServe()
	return udp.db
}

func getUDP(port string, id NodeID) *udp {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		log.Fatal(err.Error())
	}

	return newUDP(id, addr)
}

func hexToNodeID(id string) NodeID {
	byte, err := hexutil.HexToBytes(id)
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

	log.Debug("nodeid: %s", hexutil.BytesToHex(udp.self.ID.Bytes()))

	addr := getAddr(targePort)
	tid := hexToNodeID(id)

	n := NewNodeWithAddr(tid, addr)
	udp.addNode(n)

	udp.StartServe()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
