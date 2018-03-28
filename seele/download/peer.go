/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"errors"
	"math/big"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p"
)

var (
	errRecvedQuitMsg = errors.New("Recved quit msg")
	errPeerQuit      = errors.New("Peer quit")
)

type Peer interface {
	Head() (common.Hash, *big.Int)
	RequestHeadersByHashOrNumber(origin common.Hash, num uint64, amount int, reverse bool) error
	RequestBlocksByHashOrNumber(origin common.Hash, num uint64, amount int) error
}

type peerConn struct {
	peerID         string
	peer           Peer
	waitingMsgMap  map[int]chan *p2p.Message //
	lockForWaiting sync.RWMutex              //

	quitCh chan struct{}
}

func newPeerConn(p Peer, peerID string) *peerConn {
	return &peerConn{
		peerID:        peerID,
		peer:          p,
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
