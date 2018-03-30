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

func (sp *SeeleProtocol) Start() {
	event.BlockMinedEventManager.AddListener(sp.NewBlockCB)
	event.TransactionInsertedEventManager.AddListener(sp.NewTxCB)
	go sp.syncer()
}

// Stop stops protocol, called when seeleService quits.
func (sp *SeeleProtocol) Stop() {
	event.BlockMinedEventManager.RemoveListener(sp.NewBlockCB)
	event.TransactionInsertedEventManager.RemoveListener(sp.NewTxCB)
	close(sp.quitCh)
	close(sp.syncCh)
	sp.wg.Wait()
}

// syncer try to synchronise with remote peer
func (sp *SeeleProtocol) syncer() {
	defer sp.downloader.Terminate()
	defer sp.wg.Done()
	sp.wg.Add(1)

	forceSync := time.NewTicker(forceSyncInterval)
	for {
		select {
		case <-sp.syncCh:
			go sp.synchronise(sp.bestPeer())
		case <-forceSync.C:
			go sp.synchronise(sp.bestPeer())
		case <-sp.quitCh:
			return
		}
	}
}

func (sp *SeeleProtocol) bestPeer() *peer {
	sp.peersLock.RLock()
	defer sp.peersLock.RUnlock()
	var (
		bestPeer *peer
		bestTd   *big.Int
	)
	for _, p := range sp.peers {
		if _, td := p.Head(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}

func (sp *SeeleProtocol) peersWithoutBlock(hash common.Hash) []*peer {
	sp.peersLock.RLock()
	defer sp.peersLock.RUnlock()
	list := make([]*peer, 0, len(sp.peers))
	for _, p := range sp.peers {
		if !p.knownBlocks.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

func (sp *SeeleProtocol) synchronise(p *peer) {
	//TODO
}

// NewBlockCB callback when a block is mined.
func (sp *SeeleProtocol) NewBlockCB(e event.Event) {
	block := e.(*types.Block)
	hash := block.HeaderHash
	peers := sp.peersWithoutBlock(hash)

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
func (sp *SeeleProtocol) NewTxCB(e event.Event) {
	// TODO
}

// syncTransactions sends pending transactions to remote peer.
func (sp *SeeleProtocol) syncTransactions(p *peer) {
	defer sp.wg.Done()

	pending, _ := sp.txPool.Pending()
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
		go func() { resultCh <- p.sendTransactions(pending[pos : pos+needSend]) }()
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
		case <-sp.quitCh:
			break loopOut
		}
	}
	close(resultCh)
}

func (sp *SeeleProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) {
	newPeer := newPeer(SeeleVersion, p2pPeer)
	if err := newPeer.HandShake(); err != nil {
		newPeer.Disconnect(DiscHandShakeErr)
		sp.log.Error("handleAddPeer err. %s", err)
		return
	}

	// insert to peers map
	sp.peersLock.Lock()
	sp.peers[newPeer.peerID] = newPeer
	sp.peersLock.Unlock()
	sp.syncCh <- struct{}{}
}

func (sp *SeeleProtocol) handleDelPeer(p2pPeer *p2p.Peer) {
	sp.peersLock.Lock()
	peerID := fmt.Sprintf("%x", p2pPeer.Node.ID[:8])
	delete(sp.peers, peerID)
	sp.peersLock.Unlock()
}

func (sp *SeeleProtocol) handleMsg(peer *p2p.Peer, write p2p.MsgWriter, msg p2p.Message) {
	//TODO add handle msg
	sp.log.Debug("SeeleProtocol readmsg. Code[%d]", msg.Code)
	return
}
