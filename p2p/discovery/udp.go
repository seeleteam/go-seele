/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"net"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
)

type udp struct {
	conn  *net.UDPConn
	self  *Node
	table *Table

	localAddr *net.UDPAddr
}

func NewUDP(id NodeID, addr *net.UDPAddr) *udp {
	transport := &udp{
		conn:      getUDPConn(addr),
		table:     NewTable(id, addr),
		self:      NewNode(id, addr),
		localAddr: addr,
	}

	return transport
}

func (u *udp) sendPingMsg(msg *Ping, target *net.UDPAddr) {
	encoding, err := common.Encoding(msg)
	if err != nil {
		log.Info(err)
		return
	}

	buff := generateBuff(PingMsg, encoding)

	sendMsg(buff, u.localAddr, target)
}

func sendMsg(buff []byte, source, target *net.UDPAddr) {
	conn, err := net.DialUDP("udp", source, target)
	if err != nil {
		log.Info(err)
	}
	defer conn.Close()

	log.Debug("buff length:", len(buff))
	n, err := conn.Write(buff)
	if err != nil {
		log.Info(err)
	}

	log.Debug(n)
}

func (u *udp) handlePingMsg(data []byte, target *net.UDPAddr) {
	log.Debug("ping msg")

	msg := Ping{}
	err := common.Decoding(data, &msg)
	if err != nil {
		log.Info(err)
		return
	}

	response := Pong{
		ID: u.self.ID,
	}

	log.Debug("ping received from: ", hexutil.Encode(msg.ID.Bytes()))

	u.sendPongMsg(&response, target)
}

func (u *udp) sendPongMsg(msg *Pong, target *net.UDPAddr) {
	encoding, err := common.Encoding(msg)
	if err != nil {
		log.Info(err)
	}

	buff := generateBuff(PongMsg, encoding)
	sendMsg(buff, u.localAddr, target)
}

func (u *udp) handlePongMsg(data []byte, target *net.UDPAddr) {
	log.Debug("pong msg")

	msg := Pong{}
	err := common.Decoding(data, &msg)
	if err != nil {
		log.Info(err)
		return
	}

	log.Debug("pong received from: ", hexutil.Encode(msg.ID.Bytes()))
}

func (u *udp) sendFindNodeMsg(msg *FindNode, target *net.UDPAddr) {

}

func (u *udp) handleFindNodeMsg(data []byte, target *net.UDPAddr) {
	log.Debug("find node msg")
}

func (u *udp) sendNeighborsMsg(msg *Neighbors, target *net.UDPAddr) {

}

func (u *udp) handleNeighborsMsg(data []byte, target *net.UDPAddr) {
	log.Debug("neighbors msg")
}

func (u *udp) readLoop() {
	for {
		data := make([]byte, 1024)
		n, remoteAddr, err := u.conn.ReadFromUDP(data)
		if err != nil {
			log.Info(err)
		}

		log.Info("ip:", remoteAddr.IP, "port:", remoteAddr.Port, "network:", remoteAddr.Network,
			"zone:", remoteAddr.Zone)
		log.Info("n:", n)

		data = data[:n]

		if n > 0 {
			code := ByteToMsgType(data[0])

			switch code {
			case PingMsg:
				u.handlePingMsg(data[1:], remoteAddr)
			case PongMsg:
				u.handlePongMsg(data[1:], remoteAddr)
			case FindNodeMsg:
				u.handleFindNodeMsg(data[1:], remoteAddr)
			case NeighborsMsg:
				u.handleNeighborsMsg(data[1:], remoteAddr)
			}
		} else {
			log.Info("wrong length")
		}
	}
}
