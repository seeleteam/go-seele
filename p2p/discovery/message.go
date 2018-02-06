/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"net"
)

type MsgType uint8

const (
	// add msg type flag at the first byte of the message
	PingMsg      MsgType = 1
	PongMsg      MsgType = 2
	FindNodeMsg  MsgType = 3
	NeighborsMsg MsgType = 5
)

type Ping struct {
	Version uint // TODO add version check
	ID      NodeID
}

type Pong struct {
	ID NodeID
}

type FindNode struct {
	Target NodeID
}

type Neighbors struct {
	Nodes []RPCNode
}

type RPCNode struct {
	ID      NodeID
	IP      net.IP
	UDPPort uint16
}

func ByteToMsgType(byte byte) MsgType {
	return MsgType(uint8(byte))
}

func MsgTypeToByte(t MsgType) byte {
	return byte(t)
}

func generateBuff(code MsgType, encoding []byte) []byte {
	buff := []byte{MsgTypeToByte(code)}

	return append(buff, encoding...)
}
