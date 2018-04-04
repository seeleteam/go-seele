/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"errors"
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

var (
	transactionHashMsgCode uint16 = 0
	blockHashMsgCode       uint16 = 1

	transactionRequestMsgCode uint16 = 2
	transactionMsgCode        uint16 = 3

	blockRequestMsgCode uint16 = 4
	blockMsgCode        uint16 = 5

	protocolMsgCodeLength uint16 = 11
)

// SeeleProtocol service implementation of seele
type SeeleProtocol struct {
	p2p.Protocol
	peerSet *peerSet

	networkID  uint64
	downloader *downloader.Downloader
	txPool     *core.TransactionPool
	chain      *core.Blockchain

	wg     sync.WaitGroup
	quitCh chan struct{}
	syncCh chan struct{}
	log    *log.SeeleLog
}

// NewSeeleProtocol create SeeleProtocol
func NewSeeleProtocol(seele *SeeleService, log *log.SeeleLog) (s *SeeleProtocol, err error) {
	s = &SeeleProtocol{
		Protocol: p2p.Protocol{
			Name:    SeeleProtoName,
			Version: SeeleVersion,
			Length:  protocolMsgCodeLength,
		},
		networkID:  seele.networkID,
		txPool:     seele.TxPool(),
		chain:      seele.BlockChain(),
		downloader: downloader.NewDownloader(seele.BlockChain()),
		log:        log,
		quitCh:     make(chan struct{}),
		syncCh:     make(chan struct{}),

		peerSet: newPeerSet(),
	}

	s.Protocol.AddPeer = s.handleAddPeer
	s.Protocol.DeletePeer = s.handleDelPeer

	event.TransactionInsertedEventManager.AddAsyncListener(s.handleNewTx)
	event.BlockMinedEventManager.AddAsyncListener(s.handleNewMinedBlock)
	return s, nil
}

func (sp *SeeleProtocol) Start() {
	go sp.syncer()
}

// Stop stops protocol, called when seeleService quits.
func (sp *SeeleProtocol) Stop() {
	event.BlockMinedEventManager.RemoveListener(sp.handleNewMinedBlock)
	event.TransactionInsertedEventManager.RemoveListener(sp.handleNewTx)
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
			go sp.synchronise(sp.peerSet.bestPeer())
		case <-forceSync.C:
			go sp.synchronise(sp.peerSet.bestPeer())
		case <-sp.quitCh:
			return
		}
	}
}

func (sp *SeeleProtocol) synchronise(p *peer) {
	//TODO
}

// syncTransactions sends pending transactions to remote peer.
func (sp *SeeleProtocol) syncTransactions(p *peer) {
	defer sp.wg.Done()

	//pending, _ := sp.txPool.Pending()
	var pending []*types.Transaction //TODO get pending transactions from txPool
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

func (p *SeeleProtocol) handleNewTx(e event.Event) {
	p.log.Debug("find new tx")
	tx := e.(*types.Transaction)

	p.peerSet.ForEach(func(peer *peer) bool {

		err := peer.SendTransactionHash(tx.Hash)
		if err != nil {
			p.log.Warn("send transaction hash failed %s", err.Error())
		}
		return true
	})
}

func (p *SeeleProtocol) handleNewMinedBlock(e event.Event) {
	p.log.Debug("find new mined block")
	block := e.(*types.Block)

	p.peerSet.ForEach(func(peer *peer) bool {
		err := peer.SendBlockHash(block.HeaderHash)
		if err != nil {
			p.log.Warn("send mined block hash failed %s", err.Error())
		}
		return true
	})
}

func (p *SeeleProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) {
	newPeer := newPeer(SeeleVersion, p2pPeer, rw)
	if err := newPeer.HandShake(); err != nil {
		newPeer.Disconnect(DiscHandShakeErr)
		p.log.Error("handleAddPeer err. %s", err)
		return
	}

	p.peerSet.Add(newPeer)
	go p.handleMsg(newPeer)
}

func (p *SeeleProtocol) handleDelPeer(p2pPeer *p2p.Peer) {
	p.peerSet.Remove(p2pPeer.Node.ID)
}

func (p *SeeleProtocol) handleMsg(peer *peer) {
handler:
	for {
		msg, err := peer.rw.ReadMsg()
		if err != nil {
			p.log.Error("get error when read msg from %s, %s", peer.peerID.ToHex(), err)
			break
		}

		switch msg.Code {
		case transactionHashMsgCode:
			var txHash common.Hash
			err := common.Deserialize(msg.Payload, &txHash)
			if err != nil {
				p.log.Warn("deserialize transaction hash msg failed %s", err.Error())
				continue
			}

			p.log.Debug("got tx hash %s", txHash.ToHex())

			if !peer.knownTxs.Has(txHash) {
				peer.knownTxs.Add(txHash) //update peer known transaction
				err := peer.SendTransactionRequest(txHash)
				if err != nil {
					p.log.Warn("send transaction request msg failed %s", err.Error())
					break handler
				}
			}

		case transactionRequestMsgCode:
			var txHash common.Hash
			err := common.Deserialize(msg.Payload, &txHash)
			if err != nil {
				p.log.Warn("deserialize transaction request msg failed %s", err.Error())
				continue
			}

			p.log.Debug("got tx request %s", txHash.ToHex())

			tx := p.txPool.GetTransaction(txHash)
			err = peer.SendTransaction(tx)
			if err != nil {
				p.log.Warn("send transaction msg failed %s", err.Error())
				break handler
			}

		case transactionMsgCode:
			var tx types.Transaction
			err := common.Deserialize(msg.Payload, &tx)
			if err != nil {
				p.log.Warn("deserialize transaction msg failed %s", err.Error())
				continue
			}

			p.log.Debug("got transaction msg %s", tx.Hash.ToHex())
			p.txPool.AddTransaction(&tx)

		case blockHashMsgCode:
			var blockHash common.Hash
			err := common.Deserialize(msg.Payload, &blockHash)
			if err != nil {
				p.log.Warn("deserialize block hash msg failed %s", err.Error())
				continue
			}

			p.log.Debug("got block hash msg %s", blockHash.ToHex())

			if !peer.knownBlocks.Has(blockHash) {
				peer.knownBlocks.Add(blockHash)
				err := peer.SendBlockRequest(blockHash)
				if err != nil {
					p.log.Warn("send block request msg failed %s", err.Error())
					break handler
				}
			}

		case blockRequestMsgCode:
			var blockHash common.Hash
			err := common.Deserialize(msg.Payload, &blockHash)
			if err != nil {
				p.log.Warn("deserialize block request msg failed %s", err.Error())
				continue
			}

			p.log.Debug("got block request msg %s", blockHash.ToHex())
			block, err := p.chain.GetStore().GetBlock(blockHash)
			if err != nil {
				p.log.Warn("not found request block %s", err.Error())
				continue
			}

			err = peer.SendBlock(block)
			if err != nil {
				p.log.Warn("send block msg failed %s", err.Error())
			}

		case blockMsgCode:
			var block types.Block
			err := common.Deserialize(msg.Payload, &block)
			if err != nil {
				p.log.Warn("deserialize block msg failed %s", err.Error())
				continue
			}

			p.log.Debug("got block msg %s", block.HeaderHash.ToHex())
			// @todo need to make sure WriteBlock handle block fork
			p.chain.WriteBlock(&block)

		default:
			p.log.Warn("unknown code %s", msg.Code)
		}
	}

	p.peerSet.Remove(peer.peerID)
}
