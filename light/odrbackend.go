/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"
	"fmt"
	rand2 "math/rand"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

var (
	errNoMorePeers   = errors.New("No peers found")
	errServiceQuited = errors.New("Service has quited")
)

type odrBackend struct {
	lock       sync.Mutex
	msgCh      chan *p2p.Message
	quitCh     chan struct{}
	requestMap map[uint32]chan interface{}
	wg         sync.WaitGroup
	peers      *peerSet
	log        *log.SeeleLog
}

func newOdrBackend(log *log.SeeleLog) *odrBackend {
	o := &odrBackend{
		msgCh:      make(chan *p2p.Message),
		requestMap: make(map[uint32]chan interface{}),
		quitCh:     make(chan struct{}),
		log:        log,
	}

	return o
}

func (o *odrBackend) start(peers *peerSet) {
	o.peers = peers
	o.wg.Add(1)
	go o.run()
}

func (o *odrBackend) run() {
	defer o.wg.Done()
loopOut:
	for {
		select {
		case msg := <-o.msgCh:
			reqID := uint32(0)
			var reqMsg interface{}
			switch msg.Code {
			case blockMsgCode:
				var blockMsg *BlockMsgBody
				if common.Deserialize(msg.Payload, &blockMsg) != nil {
					reqMsg = blockMsg
					reqID = blockMsg.ReqID
				}
			}

			if reqID != 0 {
				o.lock.Lock()
				reqCh := o.requestMap[reqID]
				if reqCh != nil {
					delete(o.requestMap, reqID)
					// if block is retrieved correctly, sends to reqCh.
					if reqMsg.(*BlockMsgBody).Block != nil {
						reqCh <- reqMsg
					}
				}
				o.lock.Unlock()
			}
		case <-o.quitCh:
			break loopOut
		}
	}
}

func (o *odrBackend) getReqInfo() (uint32, chan interface{}, error) {
	rand2.Seed(time.Now().UnixNano())
	reqID := rand2.Uint32()
	ch := make(chan interface{})

	o.lock.Lock()
	if o.requestMap[reqID] != nil {
		panic("reqid conflicks")
	}

	o.requestMap[reqID] = ch
	o.lock.Unlock()
	return reqID, ch, nil
}

// getBlock retrieves block body from network.
func (o *odrBackend) getBlock(hash common.Hash, no uint64) (*types.Block, error) {
	peerL := o.peers.choosePeers(common.LocalShardNumber, hash)
	if len(peerL) == 0 {
		return nil, errNoMorePeers
	}

	reqID, ch, err := o.getReqInfo()
	if err != nil {
		return nil, err
	}

	// todo, add resending request to other peers if timeout occurs
	for _, p := range peerL {
		p.RequestBlocksByHashOrNumber(reqID, hash, no)
	}

	timeout := time.NewTimer(msgWaitTimeout)
	select {
	case msg := <-ch:
		reqMsg := msg.(*BlockMsgBody)
		close(ch)
		return reqMsg.Block, nil
	case <-o.quitCh:
		close(ch)
		return nil, errServiceQuited
	case <-timeout.C:
		err = fmt.Errorf("wait for msg reqid=%d timeout", reqID)
		o.lock.Lock()
		reqCh := o.requestMap[reqID]
		if reqCh != nil {
			delete(o.requestMap, reqID)
		}
		o.lock.Unlock()
		return nil, err
	}
}

func (o *odrBackend) close() {
	select {
	case <-o.quitCh:
	default:
		close(o.quitCh)
	}

	o.wg.Wait()
	close(o.msgCh)
}
