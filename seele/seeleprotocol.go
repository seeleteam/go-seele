/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"fmt"
	"sync"

	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

// SeeleProtocol service implementation of seele
type SeeleProtocol struct {
	p2p.Protocol
	peers     map[string]*peer // peers map. peerID=>peer
	peersLock sync.RWMutex
	log       *log.SeeleLog
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
		log:   log,
		peers: make(map[string]*peer),
	}
	return s, nil
}

// Run implements p2p.Protocol, called in p2p.Server.Start function
func (p *SeeleProtocol) Run() {
	p.log.Info("SeeleProtocol started...")

	for {
		select {
		case newPeer := <-p.AddPeerCh:
			go p.handleAddPeer(newPeer)
		case delPeer := <-p.DelPeerCh:
			p.handleDelPeer(delPeer)
		case msg := <-p.ReadMsgCh:
			p.handleMsg(msg)
		}
	}
}

// GetBaseProtocol implements p2p.Protocol
func (p SeeleProtocol) GetBaseProtocol() (baseProto *p2p.Protocol) {
	return &(p.Protocol)
}

func (p *SeeleProtocol) handleAddPeer(p2pPeer *p2p.Peer) {
	newPeer := newPeer(SeeleVersion, p2pPeer)
	if err := newPeer.HandShake(); err != nil {
		newPeer.Disconnect(DiscHandShakeErr)
		p.log.Error("handleAddPeer err. %s", err)
		return
	}

	// insert to peers map
	p.peersLock.Lock()
	p.peers[newPeer.peerID] = newPeer
	p.peersLock.Unlock()
}

func (p *SeeleProtocol) handleDelPeer(p2pPeer *p2p.Peer) {
	p.peersLock.Lock()
	peerID := fmt.Sprintf("%x", p2pPeer.Node.ID[:8])
	delete(p.peers, peerID)
	p.peersLock.Unlock()
}

func (p *SeeleProtocol) handleMsg(msg *p2p.Message) {
	//TODO add handle msg
	p.log.Debug("SeeleProtocol readmsg. MsgCode[%d]", msg.MsgCode)
	return
}

// Stop stop protocol, called when seeleService quits.
func (p SeeleProtocol) Stop() {
	//TODO add a quit channel
}
