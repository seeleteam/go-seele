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

	log *log.SeeleLog
}

// NewSeeleService create SeeleProtocol
func NewSeeleProtocol(networkID uint64, log *log.SeeleLog) (s *SeeleProtocol, err error) {
	s = &SeeleProtocol{
		Protocol: p2p.Protocol{
			Name:       SeeleProtoName,
			Version:    SeeleVersion,
			Length:     1,
			AddPeer:    s.handleAddPeer,
			DeletePeer: s.handleDelPeer,
			HandleMsg:  s.handleMsg,
		},
		log:   log,
		peers: make(map[string]*peer),
	}

	return s, nil
}

func (p *SeeleProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) {
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

func (p *SeeleProtocol) handleMsg(peer *p2p.Peer, write p2p.MsgWriter, msg p2p.Message) {
	//TODO add handle msg
	p.log.Debug("SeeleProtocol readmsg. Code[%d]", msg.Code)
	return
}

// Stop stops protocol, called when seeleService quits.
func (p SeeleProtocol) Stop() {
	//TODO add a quit channel
}
