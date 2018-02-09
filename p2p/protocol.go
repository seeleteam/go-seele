/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"fmt"
)

const (
	baseProtoCode uint   = 8 //start protoCode used by higher level
	ctlProtoCode  uint16 = 1 //control protoCode. For example, handshake ping pong message etc

	ctlMsgProtoHandshake uint16 = 10
	ctlMsgDiscCode       uint16 = 4
	ctlMsgPingCode       uint16 = 3
	ctlMsgPongCode       uint16 = 4
)

//Protocol base class for high level transfer protocol.
type Protocol struct {
	// Name should contain the official protocol name,
	// often a three-letter word.
	Name string

	// Version should contain the version number of the protocol.
	Version uint
	// AddPeerCh a peer joins protocol, SubProtocol should handle the channel
	AddPeerCh chan *Peer

	// DelPeerCh a peer leaves protocol
	DelPeerCh chan *Peer

	// ReadMsgCh a whole Message has recved, SubProtocol can handle as quickly as possible
	ReadMsgCh chan *Message
}

// ProtocolInterface high level protocol should implement this interface
type ProtocolInterface interface {
	Run()
	GetBaseProtocol() *Protocol
}

func (p *Protocol) cap() Cap {
	return Cap{p.Name, p.Version}
}

// Cap is the structure of a peer capability.
type Cap struct {
	Name    string
	Version uint
}

func (cap Cap) String() string {
	return fmt.Sprintf("%s/%d", cap.Name, cap.Version)
}

type capsByNameAndVersion []Cap

func (cs capsByNameAndVersion) Len() int      { return len(cs) }
func (cs capsByNameAndVersion) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs capsByNameAndVersion) Less(i, j int) bool {
	return cs[i].Name < cs[j].Name || (cs[i].Name == cs[j].Name && cs[i].Version < cs[j].Version)
}
