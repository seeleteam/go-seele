/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"errors"
	"fmt"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

var (
	errPeerNotFound = errors.New("peer not found")
	errPeerNotMatch = errors.New("peer statusData not match")
)

// SeeleProtocol service implementation of seele
type SeeleProtocol struct {
	p2p.Protocol
	peers     map[string]*peer // peers map. peerID=>peer
	peersCan  map[string]*peer // candidate peers, holding peers before handshaking
	peersLock sync.RWMutex

	networkID uint64
	txPool    *core.TransactionPool
	chain     *core.Blockchain
	log       *log.SeeleLog
}

// NewSeeleService create SeeleProtocol
func NewSeeleProtocol(seele *SeeleService, log *log.SeeleLog) (s *SeeleProtocol, err error) {
	s = &SeeleProtocol{
		Protocol: p2p.Protocol{
			Name:      SeeleProtoName,
			Version:   SeeleVersion,
			AddPeerCh: make(chan *p2p.Peer),
			DelPeerCh: make(chan *p2p.Peer),
			ReadMsgCh: make(chan *p2p.Message),
		},
		networkID: seele.networkID,
		txPool:    seele.TxPool(),
		chain:     seele.BlockChain(),
		log:       log,
		peers:     make(map[string]*peer),
		peersCan:  make(map[string]*peer),
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
	err := newPeer.SendMsg(&p.Protocol, StatusMsg, &statusData{
		ProtocolVersion: uint32(SeeleVersion),
		NetworkID:       p.networkID,
		// TODO add initialization of other variables
	})
	if err != nil {
		newPeer.Disconnect(DiscHandShakeErr)
		p.log.Error("handleAddPeer err. %s", err)
		return
	}

	// insert to peers map
	p.peersLock.Lock()
	p.peersCan[newPeer.peerID] = newPeer
	p.peersLock.Unlock()
}

func (p *SeeleProtocol) handleDelPeer(p2pPeer *p2p.Peer) {
	peerID := fmt.Sprintf("%x", p2pPeer.Node.ID[:8])
	p.delPeerByID(peerID)
}

func (p *SeeleProtocol) delPeerByID(peerID string) {
	p.peersLock.Lock()
	delete(p.peers, peerID)
	delete(p.peersCan, peerID)
	p.peersLock.Unlock()
}

func (p *SeeleProtocol) handleMsg(msg *p2p.Message) {
	//TODO add handle msg
	p.log.Debug("SeeleProtocol readmsg. MsgCode[%d]", msg.MsgCode)
	peerID := fmt.Sprintf("%x", msg.CurPeer.Node.ID[:8])
	var err error
	switch msg.MsgCode {
	case StatusMsg:
		err = p.handleStatusMsg(peerID, msg)
	default:
	}
	if err != nil {
		msg.CurPeer.Disconnect(DiscBreakOut)
	}
	return
}

func (p *SeeleProtocol) handleStatusMsg(peerID string, msg *p2p.Message) error {
	var statusMsg statusData
	err := common.Deserialize(msg.Payload, &statusMsg)
	p.peersLock.Lock()
	if err != nil {
		p.log.Info("handleStatusMsg not valid msg. %s", err)
		return err
	}

	peer, ok := p.peersCan[peerID]
	if !ok {
		p.log.Error("handleStatusMsg not found peer in peersCan")
		return errPeerNotFound
	}

	delete(p.peersCan, peerID)
	if statusMsg.NetworkID != p.networkID {
		p.log.Error("handleStatusMsg networkID not match")
		return errPeerNotMatch
	}

	// insert peer to p.peers
	p.peers[peerID] = peer
	p.log.Info("handleStatusMsg add peer to p.peers")
	return nil
}

// Stop stops protocol, called when seeleService quits.
func (p SeeleProtocol) Stop() {
	//TODO add a quit channel
}
