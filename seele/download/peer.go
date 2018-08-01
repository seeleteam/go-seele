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
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

// This timeout should not be happened, but we need to handle it in case of such errors.
const MsgWaitTimeout = time.Second * 120

var (
	errReceivedQuitMsg = errors.New("Received quit msg")
	errPeerQuit        = errors.New("Peer quit")
)

type Peer interface {
	Head() (common.Hash, *big.Int)
	RequestHeadersByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int, reverse bool) error
	RequestBlocksByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int) error
}

type peerConn struct {
	peerID         string
	peer           Peer
	waitingMsgMap  map[uint16]chan *p2p.Message //
	lockForWaiting sync.RWMutex                 //

	log    *log.SeeleLog
	quitCh chan struct{}
}

func newPeerConn(p Peer, peerID string, log *log.SeeleLog) *peerConn {
	return &peerConn{
		peerID:        peerID,
		peer:          p,
		waitingMsgMap: make(map[uint16]chan *p2p.Message),
		log:           log,
		quitCh:        make(chan struct{}),
	}
}

func (p *peerConn) close() {
	close(p.quitCh)
}

func (p *peerConn) waitMsg(magic uint32, msgCode uint16, cancelCh chan struct{}) (ret interface{}, err error) {
	rcvCh := make(chan *p2p.Message)
	p.lockForWaiting.Lock()
	p.waitingMsgMap[msgCode] = rcvCh
	p.lockForWaiting.Unlock()

Again:
	timeout := time.NewTimer(MsgWaitTimeout)
	select {
	case <-p.quitCh:
		err = errPeerQuit
	case <-cancelCh:
		err = errReceivedQuitMsg
	case msg := <-rcvCh:
		switch msgCode {
		case BlockHeadersMsg:
			var reqMsg BlockHeadersMsgBody
			if err := common.Deserialize(msg.Payload, &reqMsg); err != nil {
				goto Again
			}
			if reqMsg.Magic != magic {
				p.log.Debug("Downloader.waitMsg  BlockHeadersMsg MAGIC_NOT_MATCH msg=%s pid=%s", CodeToStr(msgCode), p.peerID)
				goto Again
			}
			ret = reqMsg.Headers
		case BlocksMsg:
			var reqMsg BlocksMsgBody
			if err := common.Deserialize(msg.Payload, &reqMsg); err != nil {
				goto Again
			}
			if reqMsg.Magic != magic {
				p.log.Debug("Downloader.waitMsg  BlocksMsg MAGIC_NOT_MATCH msg=%s pid=%s", CodeToStr(msgCode), p.peerID)
				goto Again
			}
			ret = reqMsg.Blocks
		}
	case <-timeout.C:
		err = fmt.Errorf("wait for msg %s timeout", CodeToStr(msgCode))
	}

	p.lockForWaiting.Lock()
	delete(p.waitingMsgMap, msgCode)
	p.lockForWaiting.Unlock()
	close(rcvCh)
	return
}

func (p *peerConn) deliverMsg(msgCode uint16, msg *p2p.Message) {
	defer func() {
		if recover() != nil {
			p.log.Info("peerConn.deliverMsg PANIC msg=%s pid=%s", CodeToStr(msgCode), p.peerID)
		}
	}()
	p.lockForWaiting.Lock()
	ch, ok := p.waitingMsgMap[msgCode]
	p.lockForWaiting.Unlock()
	if !ok {
		return
	}
	ch <- msg
}
