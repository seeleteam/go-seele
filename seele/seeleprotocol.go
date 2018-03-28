/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/seele/download"
)

var (
	errSyncFinished = errors.New("Sync Finished!")
)

// SeeleProtocol service implementation of seele
type SeeleProtocol struct {
	p2p.Protocol
	peers     map[string]*peer // peers map. peerID=>peer
	peersCan  map[string]*peer // candidate peers, holding peers before handshaking
	peersLock sync.RWMutex

	networkID  uint64
	downloader *downloader.Downloader
	txPool     *core.TransactionPool
	chain      *core.Blockchain

	wg     sync.WaitGroup
	quitCh chan struct{}
	syncCh chan struct{}
	log    *log.SeeleLog
}

// NewSeeleService create SeeleProtocol
func NewSeeleProtocol(seele *SeeleService, log *log.SeeleLog) (s *SeeleProtocol, err error) {
	s = &SeeleProtocol{
		Protocol: p2p.Protocol{
			Name:       SeeleProtoName,
			Version:    SeeleVersion,
			Length:     1,
			AddPeer:    s.handleAddPeer,
			DeletePeer: s.handleDelPeer,
		},
		networkID:  seele.networkID,
		txPool:     seele.TxPool(),
		chain:      seele.BlockChain(),
		downloader: downloader.NewDownloader(seele.BlockChain()),
		log:        log,
		peers:      make(map[string]*peer),
		peersCan:   make(map[string]*peer),
		quitCh:     make(chan struct{}),
		syncCh:     make(chan struct{}),
	}

	return s, nil
}

func (p *SeeleProtocol) Start() {
	event.BlockMinedEventManager.AddListener(p.NewBlockCB)
	event.TransactionInsertedEventManager.AddListener(p.NewTxCB)
	go p.syncer()
}

// Stop stops protocol, called when seeleService quits.
func (p *SeeleProtocol) Stop() {
	event.BlockMinedEventManager.RemoveListener(p.NewBlockCB)
	event.TransactionInsertedEventManager.RemoveListener(p.NewTxCB)
	close(p.quitCh)
	close(p.syncCh)
	p.wg.Wait()
}

// syncer try to synchronise with remote peer
func (p *SeeleProtocol) syncer() {
	defer p.downloader.Terminate()
	defer p.wg.Done()
	p.wg.Add(1)

	forceSync := time.NewTicker(forceSyncInterval)
	for {
		select {
		case <-p.syncCh:
			go p.synchronise(p.bestPeer())
		case <-forceSync.C:
			go p.synchronise(p.bestPeer())
		case <-p.quitCh:
			return
		}
	}
}

func (pm *SeeleProtocol) bestPeer() *peer {
	pm.peersLock.RLock()
	defer pm.peersLock.RUnlock()
	var (
		bestPeer *peer
		bestTd   *big.Int
	)
	for _, p := range pm.peers {
		if _, td := p.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}

func (pm *SeeleProtocol) peersWithoutBlock(hash common.Hash) []*peer {
	pm.peersLock.RLock()
	defer pm.peersLock.RUnlock()
	list := make([]*peer, 0, len(pm.peers))
	for _, p := range pm.peers {
		if !p.knownBlocks.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

func (pm *SeeleProtocol) synchronise(p *peer) {
	//TODO
}

// NewBlockCB callback when a block is mined.
func (pm *SeeleProtocol) NewBlockCB(e event.Event) {
	block := e.(*types.Block)
	hash := block.HeaderHash
	peers := pm.peersWithoutBlock(hash)

	// send block hash to peers first
	for _, p := range peers {
		p.sendNewBlockHash(block)
	}

	// TODO calculate td
	var td *big.Int
	// Send the block to a subset of our peers
	transfer := peers[:int(math.Sqrt(float64(len(peers))))]
	for _, p := range transfer {
		p.sendNewBlock(block, td)
	}
}

// NewTxCB callback when tx recved.
func (p *SeeleProtocol) NewTxCB(e event.Event) {
	// TODO
}

// syncTransactions sends pending transactions to remote peer.
func (pm *SeeleProtocol) syncTransactions(p *peer) {
	pending, _ := pm.txPool.Pending()
	if len(pending) == 0 {
		return
	}
	var (
		resultCh = make(chan error)
		curPos   = 0
	)

	send := func(pos int) {
		// sends txs from pos
		needSend := len(pending) - pos
		if needSend > txsyncPackSize {
			needSend = txsyncPackSize
		}

		if needSend == 0 {
			resultCh <- errSyncFinished
			return
		}
		curPos = curPos + needSend
		go func() { resultCh <- p.sendTransactions(pending[pos : pos+needSend-1]) }()
	}

	resultCh <- nil
loopOut:
	for {
		select {
		case err := <-resultCh:
			if err == errSyncFinished || err != nil {
				break loopOut
			}
			send(curPos)
		case <-pm.quitCh:
			break loopOut
		}
	}
	close(resultCh)
}

func (p *SeeleProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) {
	newPeer := newPeer(SeeleVersion, p2pPeer)
	if err := newPeer.HandShake(); err != nil {
		newPeer.Disconnect(DiscHandShakeErr)
		p.log.Error("handleAddPeer err. %s", err)
		return
	}

	// insert to peers map
	p.peersLock.Lock()
	p.peers[newPeer.peerID] = newPeer
	p.peersLock.Unlock()
	p.syncCh <- struct{}{}
}

func (p *SeeleProtocol) handleDelPeer(p2pPeer *p2p.Peer) {
	p.peersLock.Lock()
	peerID := fmt.Sprintf("%x", p2pPeer.Node.ID[:8])
	delete(p.peers, peerID)
	p.peersLock.Unlock()
}

func (p *SeeleProtocol) handleMsg(peer *p2p.Peer, write p2p.MsgWriter, msg p2p.Message) {
	//TODO add handle msg
	p.log.Debug("SeeleProtocol readmsg. Code[%d]", msg.Code)
	return
}
