/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"net"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/log"
)

func StartService(id common.Address, addr *net.UDPAddr) {
	udp := newUDP(id, addr)

	udp.StartServe()

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func getUDP(port string, id common.Address) *udp {
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		log.Fatal(err.Error())
	}

	return newUDP(id, addr)
}

func hexToAddress(id string) common.Address {
	byte, err := hexutil.HexToBytes(id)
	if err != nil {
		log.Fatal(err.Error())
	}

	nid, err := common.NewAddress(byte)
	if err != nil {
		log.Fatal(err.Error())
	}

	return nid
}

func SendPing(myId common.Address, myAddr *net.UDPAddr, id common.Address, targeAddr *net.UDPAddr) {
	udp := newUDP(myId, myAddr)

	n := NewNodeWithAddr(id, targeAddr)
	udp.addNode(n)

	udp.StartServe()

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}
