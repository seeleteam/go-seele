/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"sync"

	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

const (
	taskStatusIdle        = 0 // request task is not assigned
	taskStatusDownloading = 1 // block is downloading
	taskStatusDown        = 2 // block is downloaded
)

// masterHeadInfo header info for master peer
type masterHeadInfo struct {
	header *types.BlockHeader
	peerID string
	status int // block download status
}

// peerHeadInfo header info for ordinary peer
type peerHeadInfo struct {
	headers map[uint64]*types.BlockHeader // block no=> block header
	maxNo   uint64                        //min max blockno in headers
}

func newPeerHeadInfo() *peerHeadInfo {
	return &peerHeadInfo{
		headers: make(map[uint64]*types.BlockHeader),
	}
}

type taskMgr struct {
	fromNo, toNo     uint64                   // block number range [from, to]
	curNo            uint64                   // the smallest block number need to recv
	peersHeaderMap   map[string]*peerHeadInfo // peer's header information
	masterHeaderList []masterHeadInfo         // headers for master peer

	masterPeer string
	lock       sync.RWMutex
	log        *log.SeeleLog
}

func newTaskMgr(log *log.SeeleLog, masterPeer string, from uint64, to uint64) *taskMgr {
	t := &taskMgr{
		log:              log,
		fromNo:           from,
		toNo:             to,
		curNo:            from,
		masterPeer:       masterPeer,
		peersHeaderMap:   make(map[string]*peerHeadInfo),
		masterHeaderList: make([]masterHeadInfo, 0, to-from+1),
	}
	return t
}

// getReqHeaderInfo gets header request information, returns the start block number and amount of headers.
func (t *taskMgr) getReqHeaderInfo(peer *peerConn) (uint64, int) {
	var startNo uint64
	if peer.PeerID == t.masterPeer {
		t.lock.RLock()
		startNo = t.fromNo + uint64(len(t.masterHeaderList))
		t.lock.RUnlock()
	} else {
		t.lock.Lock()
		headInfo, ok := t.peersHeaderMap[peer.PeerID]
		if !ok {
			headInfo = newPeerHeadInfo()
			t.peersHeaderMap[peer.PeerID] = headInfo
		}
		t.lock.Unlock()

		// try remove headers that already downloaded
		for no := range headInfo.headers {
			if no < t.curNo {
				delete(headInfo.headers, no)
			}
		}

		startNo = headInfo.maxNo + 1
		if len(headInfo.headers) == 0 {
			headInfo.maxNo = 0
			startNo = t.curNo
		}
	}

	if startNo == t.toNo+1 || startNo-t.curNo >= uint64(MaxHeaderFetch) {
		// do not need to recv headers now.
		return 0, 0
	}

	amount := MaxHeaderFetch
	if uint64(MaxHeaderFetch) < (t.toNo + 1 - startNo) {
		amount = int(t.toNo - startNo + 1)
	}
	return startNo, amount
}

// getReqBlocks get block request information, returns the start block number and amount of blocks.
// should set masterHead.isDownloading = false, if send request msg error or download finished.
func (t *taskMgr) getReqBlocks(peer *peerConn) (uint64, int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	headInfo, ok := t.peersHeaderMap[peer.PeerID]
	if !ok || len(headInfo.headers) == 0 {
		return 0, 0
	}
	var startNo uint64
	for _, masterHead := range t.masterHeaderList[t.curNo-t.fromNo:] {
		if masterHead.status != taskStatusIdle {
			continue
		}
		startNo = masterHead.header.Height
		masterHead.status = taskStatusDownloading
		masterHead.peerID = peer.PeerID
		break
	}
	amount := 1

	for _, masterHead := range t.masterHeaderList[startNo+1-t.fromNo:] {
		if masterHead.status == taskStatusIdle {
			if amount < MaxBlockFetch {
				amount++
				masterHead.status = taskStatusDownloading
				masterHead.peerID = peer.PeerID
			}
			continue
		}
		break
	}

	return startNo, amount
}

// isDone returns if all blocks are downloaded
func (t *taskMgr) isDone() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.curNo == t.toNo+1
}

//deliverHeaderMsg recved header msg from peer.
func (p *taskMgr) deliverHeaderMsg(peerID string, headers []*types.BlockHeader) {
	//TODO
}

//deliverBlockMsg recved blocks msg from peer.
func (p *taskMgr) deliverBlockMsg(peerID string, headers []*types.Block) {
	//TODO need remove all flags from masterHeaders
}
