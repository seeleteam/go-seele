/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package downloader

import (
	"errors"
	"math/big"
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
	GetBlockHeadersMsg = 0x81
	BlockHeadersMsg    = 0x82
	GetBlocksMsg       = 0x83
	BlocksMsg          = 0x84
)

var (
	MaxHashFetch   = 1024 // Amount of hashes to be fetched per retrieval request
	MaxBlockFetch  = 128  // Amount of blocks to be fetched per retrieval request
	MaxHeaderFetch = 256  // Amount of block headers to be fetched per retrieval request

	MaxForkAncestry = 90000       // Maximum chain reorganisation
	peerIdleTime    = time.Second // peer's wait time for next turn if no task now

	statusNone      = 1
	statusPreparing = 2
	statusFetching  = 3
	statusCleaning  = 4
)

var (
	errIsSynchronising     = errors.New("Is synchronising")
	errPeerNotFound        = errors.New("Peer not found")
	errHashNotMatch        = errors.New("Hash not match!")
	errInvalidPacketRecved = errors.New("Invalid packet recved")
)

type Downloader struct {
	cancelCh chan struct{} // Cancel current synchronising session

	masterPeer string               // Identifier of the peer currently being used as the master
	peers      map[string]*peerConn // peers map. peerID=>peer

	syncStatus int // 0 for none; 1 for preparing; 2 for downloading; 3 for cleaning
	chain      *core.Blockchain
	sessionWG  sync.WaitGroup
	log        *log.SeeleLog
	lock       sync.RWMutex
}

func newDownloader(chain *core.Blockchain) *Downloader {
	d := &Downloader{
		peers: make(map[string]*peerConn),
		chain: chain,
	}
	d.log = log.GetLogger("download", true)
	return d
}

// Synchronise try to sync with remote peer.
func (d *Downloader) Synchronise(id string, head common.Hash, td *big.Int) error {
	localTD := d.chain.CurrentBlock().Header.Difficulty //TODO get total difficulty
	if localTD.Cmp(td) >= 0 {
		return nil
	}

	// Make sure only one routine can pass at once
	d.lock.Lock()
	if d.syncStatus != statusNone {
		d.lock.Unlock()
		return errIsSynchronising
	}
	d.syncStatus = statusPreparing
	d.lock.Unlock()

	d.cancelCh = make(chan struct{})
	d.masterPeer = id

	p, ok := d.peers[id]
	if !ok {
		return errPeerNotFound
	}

	err := d.synchronise(p, head, td)
	close(d.cancelCh)

	d.lock.Lock()
	d.syncStatus = statusNone
	d.sessionWG.Wait()
	d.lock.Unlock()
	return err
}

func (d *Downloader) synchronise(conn *peerConn, head common.Hash, td *big.Int) (err error) {
	event.BlockDownloaderEventManager.Fire(event.DownloaderStartEvent)
	defer func() {
		if err != nil {
			event.BlockDownloaderEventManager.Fire(event.DownloaderFailedEvent)
		} else {
			event.BlockDownloaderEventManager.Fire(event.DownloaderDoneEvent)
		}
	}()

	latest, err := d.fetchHeight(conn)
	if err != nil {
		return err
	}
	height := latest.Height

	origin, err := d.findAncestor(conn, height)
	if err != nil {
		return err
	}

	// need download blocks from number origin to height.
	localTD := d.chain.CurrentBlock().Header.Difficulty //TODO get total difficulty
	tm := newTaskMgr(d.log, d.masterPeer, origin, height)
	d.lock.Lock()
	d.syncStatus = statusFetching
	for _, c := range d.peers {
		_, peerTD := c.peer.Head()
		if localTD.Cmp(peerTD) >= 0 {
			continue
		}
		d.sessionWG.Add(1)

		go d.peerDownload(c, tm)
	}
	d.lock.Unlock()
	d.sessionWG.Wait()
	d.syncStatus = statusCleaning
	d.log.Info("downloader.synchronise quit!")
	return nil
}

// fetchHeight gets the latest head of peer
func (d *Downloader) fetchHeight(conn *peerConn) (*types.BlockHeader, error) {
	head, _ := conn.peer.Head()
	go conn.peer.RequestHeadersByHashOrNumber(head, 0, 1, false)
	msg, err := conn.waitMsg(BlockHeadersMsg, d.cancelCh)
	if err != nil {
		return nil, err
	}
	var headers []types.BlockHeader
	if err := common.Deserialize(msg.Payload, headers); err != nil {
		return nil, err
	}
	if len(headers) != 1 {
		return nil, errInvalidPacketRecved
	}
	if headers[0].Hash() != head {
		return nil, errHashNotMatch
	}
	return &headers[0], nil
}

// findAncestor finds the common ancestor
func (d *Downloader) findAncestor(conn *peerConn, height uint64) (uint64, error) {
	//TODO
	return 0, nil
}

// RegisterPeer add peer to download routine
func (d *Downloader) RegisterPeer(peerID string, peer Peer) {
	d.lock.Lock()
	defer d.lock.Unlock()
	newConn := newPeerConn(peer, peerID)
	d.peers[peerID] = newConn
	//TODO start download routine if session is running
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

// DeliverMsg called by seeleprotocol to deliver recved msg from network
func (d *Downloader) DeliverMsg(peerID string, msg *p2p.Message) {
	d.lock.Lock()
	peerConn, ok := d.peers[peerID]
	d.lock.Unlock()
	if !ok {
		return
	}
	peerConn.deliverMsg(int(msg.Code), msg)
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
	bMaster := (conn.peerID == d.masterPeer)
	peerID := conn.peerID
	var err error
outLoop:
	for !tm.isDone() {
		hasReqData := false
		if startNo, amount := tm.getReqHeaderInfo(conn); amount > 0 {
			hasReqData = true
			if err = conn.peer.RequestHeadersByHashOrNumber(common.Hash{}, startNo, amount, false); err != nil {
				d.log.Info("RequestHeadersByHashOrNumber err!")
				break
			}
			msg, err := conn.waitMsg(BlockHeadersMsg, d.cancelCh)
			if err != nil {
				d.log.Info("peerDownload waitMsg BlockHeadersMsg err! %s", err)
				break
			}
			var headers []*types.BlockHeader
			if err = common.Deserialize(msg.Payload, headers); err != nil {
				d.log.Info("peerDownload Deserialize err! %s", err)
				break
			}
			tm.deliverHeaderMsg(peerID, headers)
		}

		if startNo, amount := tm.getReqBlocks(conn); amount > 0 {
			hasReqData = true
			if err = conn.peer.RequestBlocksByHashOrNumber(common.Hash{}, startNo, amount); err != nil {
				d.log.Info("RequestBlocksByHashOrNumber err!")
				break
			}
			msg, err := conn.waitMsg(BlocksMsg, d.cancelCh)
			if err != nil {
				d.log.Info("peerDownload waitMsg BlocksMsg err! %s", err)
				break
			}
			var blocks []*types.Block
			if err = common.Deserialize(msg.Payload, blocks); err != nil {
				d.log.Info("peerDownload Deserialize err! %s", err)
				break
			}
			tm.deliverBlockMsg(peerID, blocks)
		}
		if hasReqData {
			continue
		}
		for {
			select {
			case <-d.cancelCh:
				break outLoop
			case <-time.After(peerIdleTime):
				break
			}
		}
	}

	if bMaster {
		d.Cancel()
	}
}
