/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"
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
	peers      peerSet
	log        *log.SeeleLog
}

func newOdrBackend(log *log.SeeleLog, peers peerSet) *odrBackend {
	o := &odrBackend{
		peers:      peers,
		msgCh:      make(chan *p2p.Message),
		requestMap: make(map[uint32]chan interface{}),
		quitCh:     make(chan struct{}),
		log:        log,
	}

	o.wg.Add(1)
	go o.run()
	return o
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
					reqCh <- reqMsg
				}
				o.lock.Unlock()
			}
		case <-o.quitCh:
			break loopOut
		}
	}
}

func (o *odrBackend) getReqInfo() (uint32, chan interface{}, []*peer, error) {
	peerL := o.peers.choosePeers()
	if len(peerL) == 0 {
		return 0, nil, nil, errNoMorePeers
	}
	rand2.Seed(time.Now().UnixNano())
	reqID := rand2.Uint32()
	ch := make(chan interface{})

	o.lock.Lock()
	if o.requestMap[reqID] != nil {
		panic("reqid conflicks")
	}

	o.requestMap[reqID] = ch
	o.lock.Unlock()
	return reqID, ch, peerL, nil
}

// getBlock retrieves block body from network.
func (o *odrBackend) getBlock(hash common.Hash, no uint64) (*types.Block, error) {
	reqID, ch, peerL, err := o.getReqInfo()
	if err != nil {
		return nil, err
	}

	// todo, add resending request to other peers if timeout occurs
	for _, p := range peerL {
		p.RequestBlocksByHashOrNumber(reqID, hash, no)
	}

	select {
	case msg := <-ch:
		reqMsg := msg.(*BlockMsgBody)
		return reqMsg.Block, nil
	case <-o.quitCh:
		return nil, errServiceQuited
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
