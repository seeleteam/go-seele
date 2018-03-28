/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"errors"
	"sync"

	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/seele"
)

var (
	errRecvedQuitMsg = errors.New("Recved quit msg")
	errPeerQuit      = errors.New("Peer quit")
)

type peerConn struct {
	*seele.Peer
	waitingMsgMap  map[int]chan *p2p.Message //
	lockForWaiting sync.RWMutex              //

	quitCh chan struct{}
}

func newPeerConn(p *seele.Peer) *peerConn {
	return &peerConn{
		Peer:          p,
		waitingMsgMap: make(map[int]chan *p2p.Message),
		quitCh:        make(chan struct{}),
	}
}

func (p *peerConn) close() {
	close(p.quitCh)
}

func (p *peerConn) waitMsg(msgCode int, cancelCh chan struct{}) (*p2p.Message, error) {
	rcvCh := make(chan *p2p.Message)
	p.lockForWaiting.Lock()
	p.waitingMsgMap[msgCode] = rcvCh
	p.lockForWaiting.Unlock()
	select {
	case <-p.quitCh:
		return nil, errPeerQuit
	case <-cancelCh:
		p.lockForWaiting.Lock()
		delete(p.waitingMsgMap, msgCode)
		p.lockForWaiting.Unlock()
		return nil, errRecvedQuitMsg
	case msg := <-rcvCh:
		p.lockForWaiting.Lock()
		delete(p.waitingMsgMap, msgCode)
		p.lockForWaiting.Unlock()
		close(rcvCh)
		return msg, nil
	}
}

func (p *peerConn) deliverMsg(msgCode int, msg *p2p.Message) {
	p.lockForWaiting.Lock()
	ch, ok := p.waitingMsgMap[msgCode]
	p.lockForWaiting.Unlock()
	if !ok {
		return
	}
	ch <- msg
}
