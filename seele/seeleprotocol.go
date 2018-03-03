/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

<<<<<<< HEAD
// SeeleProtocol service implementation of seele
=======
// SeeleProtocol
>>>>>>> Add seele service framework
type SeeleProtocol struct {
	p2p.Protocol
	maxPeers int
	peers    map[*p2p.Peer]bool
	log      *log.SeeleLog
}

// NewSeeleService create SeeleProtocol
func NewSeeleProtocol(networkID uint64, log *log.SeeleLog) (s *SeeleProtocol, err error) {
	s = &SeeleProtocol{
		Protocol: p2p.Protocol{
			Name:      SeeleProtoName,
			Version:   SeeleVersion,
			AddPeerCh: make(chan *p2p.Peer),
			DelPeerCh: make(chan *p2p.Peer),
			ReadMsgCh: make(chan *p2p.Message),
		},
<<<<<<< HEAD
		log:   log,
=======
		log:   s.log,
>>>>>>> Add seele service framework
		peers: make(map[*p2p.Peer]bool),
	}
	return s, nil
}

// Run implements p2p.Protocol, called in p2p.Server.Start function
func (p *SeeleProtocol) Run() {
	p.log.Info("SeeleProtocol started...")
<<<<<<< HEAD

=======
>>>>>>> Add seele service framework
	for {
		select {
		case peer := <-p.AddPeerCh:
			p.peers[peer] = true
		case peer := <-p.DelPeerCh:
			delete(p.peers, peer)
		case message := <-p.ReadMsgCh:
<<<<<<< HEAD
			p.log.Debug("SeeleProtocol readmsg. MsgCode[%d]", message.MsgCode)
=======
			p.log.Info("SeeleProtocol readmsg [%s]", message)
>>>>>>> Add seele service framework
		}
	}
}

// GetBaseProtocol implements p2p.Protocol
func (p SeeleProtocol) GetBaseProtocol() (baseProto *p2p.Protocol) {
	return &(p.Protocol)
}

func (p *SeeleProtocol) handleMsg(msg *p2p.Message) error {
<<<<<<< HEAD
	//TODO add handle msg
=======
>>>>>>> Add seele service framework
	return nil
}

// Stop stop protocol, called when seeleService quits.
func (p SeeleProtocol) Stop() {
	//TODO add a quit channel
}
