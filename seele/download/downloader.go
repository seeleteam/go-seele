/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"errors"
	"fmt"
	"math/big"
	rand2 "math/rand"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

const (
	GetBlockHeadersMsg uint16 = 8
	BlockHeadersMsg    uint16 = 9
	GetBlocksMsg       uint16 = 10
	BlocksPreMsg       uint16 = 11 // BlocksPreMsg is sent before BlockMsg, containing block numbers of BlockMsg.
	BlocksMsg          uint16 = 12
)

func CodeToStr(code uint16) string {
	switch code {
	case GetBlockHeadersMsg:
		return "downloader.GetBlockHeadersMsg"
	case BlockHeadersMsg:
		return "downloader.BlockHeadersMsg"
	case GetBlocksMsg:
		return "downloader.GetBlocksMsg"
	case BlocksPreMsg:
		return "downloader.BlocksPreMsg"
	case BlocksMsg:
		return "downloader.BlocksMsg"
	default:
		return "unknown"
	}
}

var (
	MaxBlockFetch  = 10  // Amount of blocks to be fetched per retrieval request
	MaxHeaderFetch = 256 // Amount of block headers to be fetched per retrieval request

	MaxForkAncestry = 90000       // Maximum chain reorganisation
	peerIdleTime    = time.Second // peer's wait time for next turn if no task now

	MaxMessageLength = 8 * 1024 * 1024
	statusNone       = 1 // no sync session
	statusPreparing  = 2 // sync session is preparing
	statusFetching   = 3 // sync session is downloading
	statusCleaning   = 4 // sync session is cleaning
)

var (
	errHashNotMatch          = errors.New("Hash not match")
	errInvalidAncestor       = errors.New("Ancestor is invalid")
	errInvalidPacketReceived = errors.New("Invalid packet received")
	ErrIsSynchronising       = errors.New("Is synchronising")
	errMaxForkAncestor       = errors.New("Can not find ancestor when reached MaxForkAncestry")
	errPeerNotFound          = errors.New("Peer not found")
	errSyncErr               = errors.New("Err occurs when syncing")
)

// Downloader sync block chain with remote peer
type Downloader struct {
	cancelCh   chan struct{}        // Cancel current synchronising session
	masterPeer string               // Identifier of the best peer
	peers      map[string]*peerConn // peers map. peerID=>peer

	syncStatus int
	tm         *taskMgr

	chain     *core.Blockchain
	sessionWG sync.WaitGroup
	log       *log.SeeleLog
	lock      sync.RWMutex
}

// BlockHeadersMsgBody represents a message struct for BlockHeadersMsg
type BlockHeadersMsgBody struct {
	Magic   uint32
	Headers []*types.BlockHeader
}

// BlocksMsgBody represents a message struct for BlocksMsg
type BlocksMsgBody struct {
	Magic  uint32
	Blocks []*types.Block
}

// NewDownloader create Downloader
func NewDownloader(chain *core.Blockchain) *Downloader {
	d := &Downloader{
		peers:      make(map[string]*peerConn),
		chain:      chain,
		syncStatus: statusNone,
	}
	d.log = log.GetLogger("download")
	return d
}

func (d *Downloader) getReadableStatus() string {
	var status string
	switch d.syncStatus {
	case statusNone:
		status = "NotSyncing"
	case statusPreparing:
		status = "Preparing"
	case statusFetching:
		status = "Downloading"
	case statusCleaning:
		status = "Cleaning"
	}
	return status
}

// getSyncInfo gets sync information of the current session.
func (d *Downloader) getSyncInfo(info *SyncInfo) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	info.Status = d.getReadableStatus()
	if d.syncStatus != statusFetching {
		return
	}

	info.Duration = fmt.Sprintf("%.2f", time.Now().Sub(d.tm.startTime).Seconds())
	info.StartNum = d.tm.fromNo
	info.Amount = d.tm.toNo - d.tm.fromNo + 1
	info.Downloaded = d.tm.downloadedNum
}

// Synchronise try to sync with remote peer.
func (d *Downloader) Synchronise(id string, head common.Hash, td *big.Int, localTD *big.Int) error {
	// Make sure only one routine can pass at once
	d.lock.Lock()
	if d.syncStatus != statusNone {
		d.lock.Unlock()
		return ErrIsSynchronising
	}
	d.syncStatus = statusPreparing
	d.cancelCh = make(chan struct{})
	d.masterPeer = id
	p, ok := d.peers[id]
	if !ok {
		close(d.cancelCh)
		d.syncStatus = statusNone
		d.lock.Unlock()
		return errPeerNotFound
	}
	d.lock.Unlock()

	err := d.doSynchronise(p, head, td, localTD)
	d.lock.Lock()
	d.syncStatus = statusNone
	d.sessionWG.Wait()
	d.cancelCh = nil
	d.lock.Unlock()
	return err
}

func (d *Downloader) doSynchronise(conn *peerConn, head common.Hash, td *big.Int, localTD *big.Int) (err error) {
	d.log.Debug("Downloader.doSynchronise start")
	event.BlockDownloaderEventManager.Fire(event.DownloaderStartEvent)
	defer func() {
		if err != nil {
			d.log.Info("download end with failed, err %s", err)
			event.BlockDownloaderEventManager.Fire(event.DownloaderFailedEvent)
		} else {
			d.log.Debug("download end success")
			event.BlockDownloaderEventManager.Fire(event.DownloaderDoneEvent)
		}
	}()

	rand2.Seed(time.Now().UnixNano())
	latest, err := d.fetchHeight(conn)
	if err != nil {
		return err
	}
	height := latest.Height

	ancestor, err := d.findCommonAncestorHeight(conn, height)
	if err != nil {
		return err
	}
	d.log.Debug("start task manager from height:%d, target height:%d", ancestor, height)
	tm := newTaskMgr(d, d.masterPeer, ancestor+1, height)
	d.tm = tm
	d.lock.Lock()
	d.syncStatus = statusFetching
	for _, pConn := range d.peers {
		_, peerTD := pConn.peer.Head()
		if localTD.Cmp(peerTD) >= 0 {
			continue
		}
		d.sessionWG.Add(1)

		go d.peerDownload(pConn, tm)
	}
	d.lock.Unlock()
	d.sessionWG.Wait()

	d.lock.Lock()
	d.syncStatus = statusCleaning
	d.lock.Unlock()
	tm.close()
	d.tm = nil
	d.log.Debug("downloader.doSynchronise quit!")

	if tm.isDone() {
		return nil
	}

	return errSyncErr
}

// fetchHeight gets the latest head of peer
func (d *Downloader) fetchHeight(conn *peerConn) (*types.BlockHeader, error) {
	head, _ := conn.peer.Head()

	magic := rand2.Uint32()
	go conn.peer.RequestHeadersByHashOrNumber(magic, head, 0, 1, false)
	msg, err := conn.waitMsg(magic, BlockHeadersMsg, d.cancelCh)
	if err != nil {
		return nil, err
	}

	headers := msg.([]*types.BlockHeader)
	if len(headers) != 1 {
		return nil, errInvalidPacketReceived
	}
	if headers[0].Hash() != head {
		return nil, errHashNotMatch
	}
	return headers[0], nil
}

// findCommonAncestorHeight finds the common ancestor height
func (d *Downloader) findCommonAncestorHeight(conn *peerConn, height uint64) (uint64, error) {
	// Get the top height
	block := d.chain.CurrentBlock()
	localHeight := block.Header.Height
	var top uint64
	if localHeight <= height {
		top = localHeight
	} else {
		top = height
	}

	if top == 0 {
		return top, nil
	}

	// get maximum chain reorganisation
	var maxFetchAncestry int
	if top >= uint64(MaxForkAncestry) {
		maxFetchAncestry = MaxForkAncestry
	} else {
		maxFetchAncestry = int(top) + 1
	}

	// Compare the peer and local block head hash and return the ancestor height
	var cmpCount = 0
	for {
		localTop := top - uint64(cmpCount)
		var fetchCount = 0
		if (maxFetchAncestry - cmpCount) >= MaxHeaderFetch {
			fetchCount = MaxHeaderFetch
		} else {
			fetchCount = maxFetchAncestry - cmpCount
		}
		if fetchCount == 0 {
			return 0, errMaxForkAncestor
		}

		// Get peer block headers
		magic := rand2.Uint32()
		go conn.peer.RequestHeadersByHashOrNumber(magic, common.EmptyHash, localTop, fetchCount, true)
		msg, err := conn.waitMsg(magic, BlockHeadersMsg, d.cancelCh)
		if err != nil {
			return 0, err
		}

		headers := msg.([]*types.BlockHeader)
		if len(headers) == 0 {
			return 0, errInvalidAncestor
		}
		cmpCount += len(headers)

		// Is ancenstor found
		for i := 0; i < len(headers); i++ {
			cmpHeight := headers[i].Height
			localHash, err := d.chain.GetStore().GetBlockHash(cmpHeight)
			if err != nil {
				return 0, err
			}
			if localHash == headers[i].Hash() {
				return cmpHeight, nil
			}
		}
	}
}

// RegisterPeer add peer to download routine
func (d *Downloader) RegisterPeer(peerID string, peer Peer) {
	d.lock.Lock()
	defer d.lock.Unlock()
	newConn := newPeerConn(peer, peerID, d.log)
	d.peers[peerID] = newConn

	if d.syncStatus == statusFetching {
		d.sessionWG.Add(1)
		go d.peerDownload(newConn, d.tm)
	}
}

// UnRegisterPeer remove peer from download routine
func (d *Downloader) UnRegisterPeer(peerID string) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if peerConn, ok := d.peers[peerID]; ok {
		peerConn.close()
		delete(d.peers, peerID)
	}
}

// DeliverMsg called by seeleprotocol to deliver received msg from network
func (d *Downloader) DeliverMsg(peerID string, msg *p2p.Message) {
	d.lock.Lock()
	peerConn, ok := d.peers[peerID]
	d.lock.Unlock()
	if !ok {
		return
	}
	peerConn.deliverMsg(msg.Code, msg)
	return
}

// Cancel cancels current session.
func (d *Downloader) Cancel() {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.cancelCh != nil {
		select {
		case <-d.cancelCh:
		default:
			close(d.cancelCh)
		}
	}
}

// Terminate close Downloader, cannot called anymore.
func (d *Downloader) Terminate() {
	d.Cancel()
	d.sessionWG.Wait()
	// TODO release variables if needed
}

// peerDownload peer download routine
func (d *Downloader) peerDownload(conn *peerConn, tm *taskMgr) {
	defer d.sessionWG.Done()
	d.log.Debug("Downloader.peerDownload start")
	bMaster := (conn.peerID == d.masterPeer)
	peerID := conn.peerID
	var err error
outLoop:
	for !tm.isDone() {
		hasReqData := false
		if startNo, amount := tm.getReqHeaderInfo(conn); amount > 0 {
			d.log.Debug("tm.getReqHeaderInfo. startNo:%d amount:%d", startNo, amount)
			hasReqData = true

			d.log.Debug("request header by number. start %d, amount %d", startNo, amount)
			magic := rand2.Uint32()
			if err = conn.peer.RequestHeadersByHashOrNumber(magic, common.Hash{}, startNo, amount, false); err != nil {
				d.log.Warn("RequestHeadersByHashOrNumber err! %s pid=%s", err, peerID)
				break
			}
			msg, err := conn.waitMsg(magic, BlockHeadersMsg, d.cancelCh)
			if err != nil {
				d.log.Warn("peerDownload waitMsg BlockHeadersMsg err! %s", err)
				break
			}

			headers := msg.([]*types.BlockHeader)
			startHeight := uint64(0)
			endHeight := uint64(0)
			if len(headers) > 0 {
				startHeight = headers[0].Height
				endHeight = headers[len(headers)-1].Height
			}
			d.log.Debug("got block header msg length %d. start %d, end %d", len(headers), startHeight, endHeight)

			if err = tm.deliverHeaderMsg(peerID, headers); err != nil {
				d.log.Warn("peerDownload deliverHeaderMsg err! %s", err)
				break
			}

			d.log.Debug("get request header info success")
		}

		if startNo, amount := tm.getReqBlocks(conn); amount > 0 {
			d.log.Debug("download.peerdown getReqBlocks startNo=%d amount=%d", startNo, amount)
			hasReqData = true

			d.log.Debug("request block by number. start %d, amount %d", startNo, amount)
			magic := rand2.Uint32()
			if err = conn.peer.RequestBlocksByHashOrNumber(magic, common.Hash{}, startNo, amount); err != nil {
				d.log.Warn("RequestBlocksByHashOrNumber err! %s", err)
				break
			}

			msg, err := conn.waitMsg(magic, BlocksMsg, d.cancelCh)
			if err != nil {
				d.log.Warn("peerDownload waitMsg BlocksMsg err! %s", err)
				break
			}

			blocks := msg.([]*types.Block)
			startHeight := uint64(0)
			endHeight := uint64(0)
			if len(blocks) > 0 {
				startHeight = blocks[0].Header.Height
				endHeight = blocks[len(blocks)-1].Header.Height
			}
			d.log.Debug("got blocks message length %d. start %d, end %d", len(blocks), startHeight, endHeight)

			tm.deliverBlockMsg(peerID, blocks)
			d.log.Debug("get request blocks success")
		}

		if hasReqData {
			d.log.Debug("got request data, continue to request")
			continue
		}

	outFor:
		for {
			select {
			case <-d.cancelCh:
				break outLoop
			case <-conn.quitCh:
				break outLoop
			case <-time.After(peerIdleTime):
				d.log.Debug("peerDownload peerIdleTime timeout")
				break outFor
			}
		}
	}

	tm.onPeerQuit(peerID)
	if bMaster {
		d.Cancel()
	}
	d.log.Debug("Downloader.peerDownload end")
}

// processBlocks writes blocks to the blockchain.
func (d *Downloader) processBlocks(headInfos []*downloadInfo) {
	for _, h := range headInfos {
		d.log.Debug("height:%d hash:%s <- preHash:%s ", h.block.Header.Height, h.block.HeaderHash.ToHex(), h.block.Header.PreviousBlockHash.ToHex())
	}

	for _, h := range headInfos {
		d.log.Debug("d.processBlock %d", h.block.Header.Height)
		if err := d.chain.WriteBlock(h.block); err != nil && err != core.ErrBlockAlreadyExists {
			d.log.Error("downloader processBlocks err. %s", err)
			d.Cancel()
			break
		}
		h.status = taskStatusProcessed
	}
}
