/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"fmt"
	rand2 "math/rand"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/store"
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
	requestMap map[uint32]chan odrResponse
	wg         sync.WaitGroup
	peers      *peerSet
	bcStore    store.BlockchainStore // used to validate the retrieved ODR object.
	log        *log.SeeleLog

	shard uint
}

func newOdrBackend(bcStore store.BlockchainStore, shard uint) *odrBackend {
	o := &odrBackend{
		msgCh:      make(chan *p2p.Message),
		requestMap: make(map[uint32]chan odrResponse),
		quitCh:     make(chan struct{}),
		bcStore:    bcStore,
		log:        log.GetLogger("odrBackend"),
		shard:      shard,
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
			o.handleResponse(msg)
		case <-o.quitCh:
			break loopOut
		}
	}
}

func (o *odrBackend) handleResponse(msg *p2p.Message) {
	factory, ok := odrResponseFactories[msg.Code]
	if !ok {
		return
	}

	response := factory()
	if err := common.Deserialize(msg.Payload, response); err != nil {
		o.log.Error("Failed to deserialize ODR response, code = %s, error = %s", codeToStr(msg.Code), err)
		return
	}

	o.lock.Lock()
	defer o.lock.Unlock()

	if reqCh, ok := o.requestMap[response.getRequestID()]; ok {
		delete(o.requestMap, response.getRequestID())
		reqCh <- response
	}
}

func (o *odrBackend) getReqInfo(filter peerFilter) (uint32, chan odrResponse, []*peer, error) {
	peerL := o.peers.choosePeers(filter)
	if len(peerL) == 0 {
		return 0, nil, nil, errNoMorePeers
	}
	rand2.Seed(time.Now().UnixNano())
	reqID := rand2.Uint32()
	ch := make(chan odrResponse)

	o.lock.Lock()
	if o.requestMap[reqID] != nil {
		panic("reqid conflicks")
	}

	o.requestMap[reqID] = ch
	o.lock.Unlock()
	return reqID, ch, peerL, nil
}

// retrieve retrieves the requested ODR object from remote peer.
func (o *odrBackend) retrieve(request odrRequest) (odrResponse, error) {
	return o.retrieveWithFilter(request, peerFilter{})
}

// retrieve retrieves the requested ODR object from remote peer with specified peer filter.
func (o *odrBackend) retrieveWithFilter(request odrRequest, filter peerFilter) (odrResponse, error) {
	reqID, ch, peerL, err := o.getReqInfo(filter)
	if err != nil {
		return nil, err
	}
	defer close(ch)

	request.setRequestID(reqID)
	code, payload := request.code(), common.SerializePanic(request)
	for _, p := range peerL {
		o.log.Debug("peer send request, code = %s, payloadSizeBytes = %v", codeToStr(code), len(payload))
		if err = p2p.SendMessage(p.rw, code, payload); err != nil {
			o.log.Info("Failed to send message with peer %v", p)
			return nil, errors.NewStackedErrorf(err, "failed to send P2P message")
		}
	}

	timeout := time.NewTimer(msgWaitTimeout)
	defer timeout.Stop()

	select {
	case resp := <-ch:
		if err := resp.getError(); err != nil {
			return nil, errors.NewStackedError(err, "failed to handle ODR request on server side")
		}

		if err := resp.validate(request, o.bcStore); err != nil {
			return nil, errors.NewStackedError(err, "failed to valdiate ODR response")
		}

		return resp, nil
	case <-o.quitCh:
		return nil, errServiceQuited
	case <-timeout.C:
		o.lock.Lock()
		delete(o.requestMap, reqID)
		o.lock.Unlock()
		return nil, fmt.Errorf("wait for msg reqid=%d timeout", reqID)
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
