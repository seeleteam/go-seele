/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"github.com/seeleteam/go-seele/common"
	"net"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

const (
	PINGPONGINTERVER = 500 * time.Millisecond // sleep between ping pong
	DISCOVERYINTERVER = 10 * time.Second // sleep between discovery
)

func StartServer(port string) {
	udp := getUDP(port)
	log.Debug("nodeid:" + common.BytesToHex(udp.self.ID.Bytes()))

	udp.StartServe()

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func getUDP(port string) *udp {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	//TODO generate key for test
	keypair, err := crypto.GenerateKey()
	if err != nil {
		log.Info(err)
	}

	buff := crypto.FromECDSAPub(&keypair.PublicKey)

	id, err := BytesTOID(buff[1:])
	if err != nil {
		log.Fatal(err)
	}

	return NewUDP(id, addr)
}

func SendPing(port, id, targePort string) {
	udp := getUDP(port)

	log.Debug("nodeid: " + common.BytesToHex(udp.self.ID.Bytes()))

	addr := getAddr(targePort)
	byte, err := common.HexToBytes(id)
	if err != nil {
		log.Fatal(err)
	}

	nid, err := BytesTOID(byte)
	if err != nil {
		log.Fatal(err)
	}

	n := NewNodeWithAddr(nid, addr)
	udp.table.addNode(n)
	udp.db.add(n.sha, n)

	udp.StartServe()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
