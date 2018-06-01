/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/p2p"
)

const MsgWaitTimeout = time.Second * 120

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
	waitingMsgMap  map[uint16]chan *p2p.Message //
	lockForWaiting sync.RWMutex                 //

	quitCh chan struct{}
}

func newPeerConn(p Peer, peerID string) *peerConn {
	return &peerConn{
		peerID:        peerID,
		peer:          p,
		waitingMsgMap: make(map[uint16]chan *p2p.Message),
		quitCh:        make(chan struct{}),
	}
}

func (p *peerConn) close() {
	close(p.quitCh)
}

func (p *peerConn) waitMsg(msgCode uint16, cancelCh chan struct{}) (*p2p.Message, error) {
	rcvCh := make(chan *p2p.Message)
	p.lockForWaiting.Lock()
	p.waitingMsgMap[msgCode] = rcvCh
	p.lockForWaiting.Unlock()

	timeout := time.NewTimer(MsgWaitTimeout)
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
	case <- timeout.C:
		return nil, fmt.Errorf("wait for msg %s timeout", codeToStr(msgCode))
	}
}

func (p *peerConn) deliverMsg(msgCode uint16, msg *p2p.Message) {
	p.lockForWaiting.Lock()
	ch, ok := p.waitingMsgMap[msgCode]
	p.lockForWaiting.Unlock()
	if !ok {
		return
	}
	ch <- msg
}
