/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"net"
	"sync"
	_ "sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	_ "github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

func StartServer(port string) {
	udp := getUDP(port)
	log.Debug("nodeid:" + hexutil.Encode(udp.self.ID.Bytes()))

	udp.readLoop()
}

func getUDP(port string) *udp {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

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

func SendPing(port string, target string) {
	udp := getUDP(port)

	go udp.readLoop()

	log.Debug("nodeid: " + hexutil.Encode(udp.self.ID.Bytes()))

	addr, err := net.ResolveUDPAddr("udp", ":"+target)
	if err != nil {
		log.Fatal(err)
	}

	msg := &Ping{
		ID: udp.self.ID,
	}

	udp.sendPingMsg(msg, addr)

	var wg sync.WaitGroup
	wg.Add(1)

	wg.Wait()
}
