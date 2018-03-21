/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"fmt"
)

const (
	baseProtoCode uint16 = 16 //start protoCode used by higher level
	ctlProtoCode  uint16 = 1  //control protoCode. For example, handshake ping pong message etc
)

//Protocol base class for high level transfer protocol.
type Protocol struct {
	// Name should contain the official protocol name,
	// often a three-letter word.
	Name string

	// Version should contain the version number of the protocol.
	Version uint

	// Length should contain the number of message codes used by the protocol.
	Length uint16

	// Run is called in a new groutine when the protocol has been
	// negotiated with a peer. It should read and write messages from
	// rw. The Payload for each message must be fully consumed.
	//
	// The peer connection is closed when Start returns. It should return
	// any protocol-level error (such as an I/O error) that is
	// encountered.
	run func(peer *Peer, rw MsgReadWriter) error

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
