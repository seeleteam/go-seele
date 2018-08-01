/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"errors"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

const (
	taskStatusIdle           = 0 // request task is not assigned
	taskStatusDownloading    = 1 // block is downloading
	taskStatusWaitProcessing = 2 // block is downloaded, needs to process
	taskStatusProcessed      = 3 // block is written to chain

	maxBlocksWaiting = 1024 // max blocks waiting to download
)

var (
	errMasterHeadersNotMatch = errors.New("Master headers not match")
	errHeadInfoNotFound      = errors.New("Header info not found")
)

// downloadInfo header info for master peer
type downloadInfo struct {
	header *types.BlockHeader
	block  *types.Block
	peerID string
	status int // block download status
}

// peerHeadInfo header info for ordinary peer
type peerHeadInfo struct {
	headers map[uint64]*types.BlockHeader // block height=> block header
	maxNo   uint64                        //max block height in headers
}

func newPeerHeadInfo() *peerHeadInfo {
	return &peerHeadInfo{
		headers: make(map[uint64]*types.BlockHeader),
	}
}

type taskMgr struct {
	downloader       *Downloader
	fromNo, toNo     uint64 // block number range [from, to]
	curNo            uint64 // the smallest block number need to recv
	downloadedNum    uint64
	peersHeaderMap   map[string]*peerHeadInfo // peer's header information
	downloadInfoList []*downloadInfo          // download process info

	masterPeer string
	lock       sync.RWMutex
	quitCh     chan struct{}
	wg         sync.WaitGroup
	log        *log.SeeleLog
	startTime  time.Time
}

func newTaskMgr(d *Downloader, masterPeer string, from uint64, to uint64) *taskMgr {
	t := &taskMgr{
		log:              d.log,
		downloader:       d,
		fromNo:           from,
		toNo:             to,
		curNo:            from,
		downloadedNum:    0,
		masterPeer:       masterPeer,
		startTime:        time.Now(),
		peersHeaderMap:   make(map[string]*peerHeadInfo),
		downloadInfoList: make([]*downloadInfo, 0, to-from+1),
		quitCh:           make(chan struct{}),
	}
	t.wg.Add(1)
	go t.run()
	return t
}

func (t *taskMgr) run() {
	defer t.wg.Done()
loopOut:
	for {
		t.lock.Lock()
		startPos := int(t.curNo - t.fromNo)
		num := 0
		for (startPos+num < len(t.downloadInfoList)) && (t.downloadInfoList[startPos+num].status == taskStatusWaitProcessing) {
			num = num + 1
		}

		results := t.downloadInfoList[startPos : startPos+num]
		t.curNo = t.curNo + uint64(num)
		t.lock.Unlock()
		t.downloader.processBlocks(results)

		select {
		case <-time.After(time.Second):
		case <-t.quitCh:
			break loopOut
		}
	}
}

func (t *taskMgr) close() {
	select {
	case <-t.quitCh:
	default:
		close(t.quitCh)
	}
	t.wg.Wait()
}

// getReqHeaderInfo gets header request information, returns the start block number and amount of headers.
func (t *taskMgr) getReqHeaderInfo(conn *peerConn) (uint64, int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	headInfo, ok := t.peersHeaderMap[conn.peerID]
	if !ok {
		headInfo = newPeerHeadInfo()
		t.peersHeaderMap[conn.peerID] = headInfo
		t.log.Debug("getReqHeaderInfo. create headInfo for peer: %s", conn.peerID)
	}

	// try remove headers that already downloaded
	for no := range headInfo.headers {
		if no < t.curNo {
			delete(headInfo.headers, no)
		}
	}

	var startNo uint64
	if conn.peerID == t.masterPeer {
		startNo = t.fromNo + uint64(len(t.downloadInfoList))
		if startNo-t.curNo > maxBlocksWaiting {
			return 0, 0
		}
	} else {
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
	if uint64(MaxHeaderFetch) > (t.toNo + 1 - startNo) {
		amount = int(t.toNo - startNo + 1)
	}
	return startNo, amount
}

// getReqBlocks get block request information, returns the start block number and amount of blocks.
// should set masterHead.isDownloading = false, if send request msg error or download finished.
func (t *taskMgr) getReqBlocks(conn *peerConn) (uint64, int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	headInfo, ok := t.peersHeaderMap[conn.peerID]
	if !ok || len(headInfo.headers) == 0 {
		return 0, 0
	}
	var startNo uint64
	var amount int
	// find the first block that not requested yet and exists in conn
	for _, masterHead := range t.downloadInfoList[t.curNo-t.fromNo:] {
		if masterHead.status != taskStatusIdle {
			continue
		}
		curHeight := masterHead.header.Height
		peerHead, ok := headInfo.headers[curHeight]
		if !ok || peerHead.Hash() != masterHead.header.Hash() {
			continue
		}

		startNo = masterHead.header.Height
		masterHead.status = taskStatusDownloading
		masterHead.peerID = conn.peerID
		amount = 1
		break
	}

	if amount == 0 {
		return 0, 0
	}

	for _, masterHead := range t.downloadInfoList[startNo+1-t.fromNo:] {
		if masterHead.status == taskStatusIdle {
			peerHead, ok := headInfo.headers[startNo+uint64(amount)]
			// if block is not found in headers or hash not match, then breaks the loop
			if !ok || peerHead.Hash() != masterHead.header.Hash() {
				break
			}
			if amount < MaxBlockFetch {
				amount++
				masterHead.status = taskStatusDownloading
				masterHead.peerID = conn.peerID
			} else {
				break
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
	t.log.Debug("task is done check cur:%d, target:%d", t.curNo, t.toNo)
	return t.curNo == t.toNo+1
}

// onPeerQuit needs to remove tasks assigned to peer
func (t *taskMgr) onPeerQuit(peerID string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, masterHead := range t.downloadInfoList[t.curNo-t.fromNo:] {
		if masterHead.status == taskStatusDownloading && masterHead.peerID == peerID {
			masterHead.peerID = ""
			masterHead.status = taskStatusIdle
		}
	}
}

// deliverHeaderMsg received header msg from peer.
func (t *taskMgr) deliverHeaderMsg(peerID string, headers []*types.BlockHeader) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if len(headers) == 0 {
		t.log.Debug("get block header msg with empty header info")
		return nil
	}

	if peerID == t.masterPeer {
		lastNo := t.fromNo + uint64(len(t.downloadInfoList))
		t.log.Debug("masterPeer deliverHeaderMsg. lastNo=%d fromNo:%d header.height:%d", lastNo, t.fromNo, headers[0].Height)
		if lastNo != headers[0].Height {
			return errMasterHeadersNotMatch
		}
		for _, h := range headers {
			t.downloadInfoList = append(t.downloadInfoList, &downloadInfo{
				header: h,
				status: taskStatusIdle,
			})
		}
	}

	headInfo, ok := t.peersHeaderMap[peerID]
	if !ok {
		return errHeadInfoNotFound
	}

	for _, h := range headers {
		headInfo.headers[h.Height] = h
		if headInfo.maxNo < h.Height {
			headInfo.maxNo = h.Height
		}
	}

	return nil
}

// deliverBlockMsg received blocks msg from peer.
func (t *taskMgr) deliverBlockMsg(peerID string, blocks []*types.Block) {
	t.lock.Lock()
	defer t.lock.Unlock()
	for _, b := range blocks {
		headInfo := t.downloadInfoList[int(b.Header.Height-t.fromNo)]
		if headInfo.peerID != peerID {
			t.log.Info("Received block from different peer, discard this block. peerID=%s", peerID)
			continue
		}

		headInfo.block = b
		headInfo.status = taskStatusWaitProcessing
		t.downloadedNum++
	}
}
