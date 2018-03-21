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

	// AddPeer find a new peer will call this method
	AddPeer func(peer *Peer, rw MsgReadWriter)

	// DeletePeer this method will be called when a peer is disconnected
	DeletePeer func(peer *Peer)

	// HandleMsg get a new peers msg, you could call write to response
	HandleMsg func(peer *Peer, write MsgWriter, msg Message)
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
