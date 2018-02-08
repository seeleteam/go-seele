/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"fmt"
)

const (
	baseProtoCode uint   = 8
	ctlProtoCode  uint16 = 1

	ctlMsgProtoHandshake uint16 = 10
	ctlMsgDiscCode       uint16 = 4
	ctlMsgPingCode       uint16 = 3
	ctlMsgPongCode       uint16 = 4
)

//Protocol base class
type Protocol struct {
	// Protocol represents a P2P subprotocol implementation.
	// Name should contain the official protocol name,
	// often a three-letter word.
	Name string //see dwn gss

	// Version should contain the version number of the protocol.
	Version uint

	AddPeerCh chan *Peer
	DelPeerCh chan *Peer
	ReadMsgCh chan *Message
}

type ProtocolInterface interface {
	Run()
	GetBaseProtocol() *Protocol
}

/*
func (p *Protocol) Start() {
	addPeerCh = make(chan *Peer)
	delPeerCh = make(chan *Peer)
	readMsgCh = make(chan *Message)
	go p.Run()
	return
}
*/
func (p *Protocol) cap() Cap {
	return Cap{p.Name, p.Version}
}

// Cap is the structure of a peer capability.
type Cap struct {
	Name    string
	Version uint
}

/*
// RlpData for RlpData
func (cap Cap) RlpData() interface{} {
	return []interface{}{cap.Name, cap.Version}
}
*/
func (cap Cap) String() string {
	return fmt.Sprintf("%s/%d", cap.Name, cap.Version)
}

type capsByNameAndVersion []Cap

func (cs capsByNameAndVersion) Len() int      { return len(cs) }
func (cs capsByNameAndVersion) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }
func (cs capsByNameAndVersion) Less(i, j int) bool {
	return cs[i].Name < cs[j].Name || (cs[i].Name == cs[j].Name && cs[i].Version < cs[j].Version)
}
