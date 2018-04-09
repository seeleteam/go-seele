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
	transactionsMsgCode       uint16 = 3

	blockRequestMsgCode uint16 = 4
	blockMsgCode        uint16 = 5

	statusDataMsgCode      uint16 = 6
	statusChainHeadMsgCode uint16 = 7
	//transactionMsgCode uint16 = 8
	protocolMsgCodeLength uint16 = 13
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
	if p == nil {
		return
	}

	block, _ := sp.chain.CurrentBlock()
	localTD, err := sp.chain.GetStore().GetBlockTotalDifficulty(block.HeaderHash)
	if err != nil {
		sp.log.Error("sp.synchronise GetBlockTotalDifficulty err.[%s]", err)
		return
	}
	pHead, pTd := p.Head()

	// if total difficulty is not smaller than remote peer td, then do not need synchronise.
	if localTD.Cmp(pTd) >= 0 {
		return
	}

	err = sp.downloader.Synchronise(p.peerStrID, pHead, pTd, localTD)
	if err != nil {
		sp.log.Error("synchronise err. %s", err)
		return
	}

	//broadcast chain head
	sp.broadcastChainHead()
}

func (sp *SeeleProtocol) broadcastChainHead() {
	block, _ := sp.chain.CurrentBlock()
	head := block.HeaderHash
	localTD, err := sp.chain.GetStore().GetBlockTotalDifficulty(head)
	if err != nil {
		return
	}
	status := &chainHeadStatus{
		TD:           localTD,
		CurrentBlock: head,
	}
	sp.peerSet.ForEach(func(peer *peer) bool {
		err := peer.sendHeadStatus(status)
		if err != nil {
			sp.log.Warn("send transaction hash failed %s", err.Error())
		}
		return true
	})
}

// syncTransactions sends pending transactions to remote peer.
func (sp *SeeleProtocol) syncTransactions(p *peer) {
	defer sp.wg.Done()

	txs := sp.txPool.GetProcessableTransactions()
	pending := make([]*types.Transaction, 0)
	for _, value := range txs {
		pending = append(pending, value...)
	}

	if len(pending) == 0 {
		return
	}
	var (
		resultCh = make(chan error, 1)
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

		if err := peer.sendTransaction(tx); err != nil {
			p.log.Warn("send transaction failed %s", err.Error())
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

	p.broadcastChainHead()
}

func (p *SeeleProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) {
	newPeer := newPeer(SeeleVersion, p2pPeer, rw)

	block, _ := p.chain.CurrentBlock()
	head := block.HeaderHash
	localTD, err := p.chain.GetStore().GetBlockTotalDifficulty(head)
	if err != nil {
		return
	}
	//genenis TODO get genenis from blockchain
	if err := newPeer.HandShake(p.networkID, localTD, head, common.EmptyHash); err != nil {
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
			p.log.Error("get error when read msg from %s, %s", peer.peerID, err)
			break
		}

		switch msg.Code {
		case transactionsMsgCode:
			var txs []*types.Transaction
			err := common.Deserialize(msg.Payload, &txs)
			if err != nil {
				p.log.Warn("deserialize transaction msg failed %s", err.Error())
				break
			}

			p.log.Debug("sendTransactionsMsgCode recved %s")
			for _, tx := range txs {
				p.txPool.AddTransaction(tx)
				peer.markTransaction(tx.Hash)
			}

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

		case downloader.GetBlockHeadersMsg:
			var query blockHeadersQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				p.log.Error("deserialize downloader.GetBlockHeadersMsg failed, quit! %s", err.Error())
				break
			}
			p.log.Debug("Recved downloader.GetBlockHeadersMsg")
			var headL []*types.BlockHeader
			var head *types.BlockHeader
			orgNum := query.Number
			if query.Hash != common.EmptyHash {
				if head, err = p.chain.GetStore().GetBlockHeader(query.Hash); err != nil {
					p.log.Error("HandleMsg GetBlockHeader err. %s", err)
					break
				}
				orgNum = head.Height
			}

			for cnt := 0; cnt < query.Amount; cnt++ {
				var curNum uint64
				if query.Reverse {
					curNum = orgNum - uint64(cnt)
				} else {
					curNum = orgNum + uint64(cnt)
				}

				hash, _ := p.chain.GetStore().GetBlockHash(curNum)
				if head, err = p.chain.GetStore().GetBlockHeader(hash); err != nil {
					p.log.Error("HandleMsg GetBlockHeader err. %s", err)
					break handler
				}
				headL = append(headL, head)
			}

			if err = peer.sendBlockHeaders(headL); err != nil {
				p.log.Error("HandleMsg sendBlockHeaders err. %s", err)
				break handler
			}
			p.log.Debug("send downloader.sendBlockHeaders")

		case downloader.GetBlocksMsg:
			p.log.Debug("Recved downloader.GetBlocksMsg")
			var query blocksQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				p.log.Error("deserialize downloader.GetBlocksMsg failed, quit! %s", err.Error())
				break
			}

			var blocksL []*types.Block
			var head *types.BlockHeader
			var block *types.Block
			orgNum := query.Number
			if query.Hash != common.EmptyHash {
				if head, err = p.chain.GetStore().GetBlockHeader(query.Hash); err != nil {
					p.log.Error("HandleMsg GetBlockHeader err. %s", err)
					break
				}
				orgNum = head.Height
			}

			totalLen := 0
			var numL []uint64
			for cnt := 0; cnt < query.Amount; cnt++ {
				curNum := orgNum + uint64(cnt)
				hash, _ := p.chain.GetStore().GetBlockHash(curNum)
				if block, err = p.chain.GetStore().GetBlock(hash); err != nil {
					p.log.Error("HandleMsg GetBlocksMsg p.chain.GetStore().GetBlock err. %s", err)
					break handler
				}

				curLen := len(common.SerializePanic(block))
				if totalLen > 0 && (totalLen+curLen) > downloader.MaxMessageLength {
					break
				}
				totalLen += curLen
				blocksL = append(blocksL, block)
				numL = append(numL, curNum)
			}

			if err = peer.sendPreBlocksMsg(numL); err != nil {
				p.log.Error("HandleMsg GetBlocksMsg sendPreBlocksMsg err. %s", err)
				break handler
			}

			if err = peer.sendBlocks(blocksL); err != nil {
				p.log.Error("HandleMsg GetBlocksMsg sendBlocks err. %s", err)
				break handler
			}
			p.log.Debug("send downloader.sendBlockHeaders")

		case downloader.BlockHeadersMsg, downloader.BlocksPreMsg, downloader.BlocksMsg:
			p.log.Debug("Recved downloader Msg. %d", msg.Code)
			p.downloader.DeliverMsg(peer.peerStrID, &msg)

		case statusChainHeadMsgCode:
			var status chainHeadStatus
			err := common.Deserialize(msg.Payload, &status)
			if err != nil {
				p.log.Error("deserialize statusChainHeadMsgCode failed, quit! %s", err.Error())
				break
			}

			p.log.Debug("Recved statusChainHeadMsgCode")
			peer.SetHead(status.CurrentBlock, status.TD)
			p.syncCh <- struct{}{}

		default:
			p.log.Warn("unknown code %s", msg.Code)
		}
	}

	p.peerSet.Remove(peer.peerID)
}
