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
	transactionHashMsgCode    uint16 = 0
	transactionRequestMsgCode uint16 = 1
	transactionsMsgCode       uint16 = 2
	blockHashMsgCode          uint16 = 3
	blockRequestMsgCode       uint16 = 4
	blockMsgCode              uint16 = 5

	statusDataMsgCode      uint16 = 6
	statusChainHeadMsgCode uint16 = 7

	protocolMsgCodeLength uint16 = 13
)

func codeToStr(code uint16) string {
	switch code {
	case transactionHashMsgCode:
		return "transactionHashMsgCode"
	case transactionRequestMsgCode:
		return "transactionRequestMsgCode"
	case transactionsMsgCode:
		return "transactionsMsgCode"
	case blockHashMsgCode:
		return "blockHashMsgCode"
	case blockRequestMsgCode:
		return "blockRequestMsgCode"
	case blockMsgCode:
		return "blockMsgCode"
	case statusDataMsgCode:
		return "statusDataMsgCode"
	case statusChainHeadMsgCode:
		return "statusChainHeadMsgCode"
	}

	return downloader.CodeToStr(code)
}

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

// Downloader return a pointer of the downloader
func (s *SeeleProtocol) Downloader() *downloader.Downloader { return s.downloader }

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
	s.Protocol.GetPeer = s.handleGetPeer

	event.TransactionInsertedEventManager.AddAsyncListener(s.handleNewTx)
	event.BlockMinedEventManager.AddAsyncListener(s.handleNewMinedBlock)
	return s, nil
}

func (sp *SeeleProtocol) Start() {
	sp.log.Debug("SeeleProtocol.Start called!")
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
			go sp.synchronise(sp.peerSet.bestPeer(common.LocalShardNumber))
		case <-forceSync.C:
			go sp.synchronise(sp.peerSet.bestPeer(common.LocalShardNumber))
		case <-sp.quitCh:
			return
		}
	}
}

func (sp *SeeleProtocol) synchronise(p *peer) {
	if p == nil {
		return
	}

	if common.PrintExplosionLog {
		sp.log.Debug("sp.synchronise called.")
	}

	block := sp.chain.CurrentBlock()
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
		if err == downloader.ErrIsSynchronising {
			sp.log.Info("exit synchronise as it is already running.")
		} else {
			sp.log.Error("synchronise err. %s", err)
		}
		return
	}

	//broadcast chain head
	sp.broadcastChainHead()
}

func (sp *SeeleProtocol) broadcastChainHead() {
	block := sp.chain.CurrentBlock()
	head := block.HeaderHash
	localTD, err := sp.chain.GetStore().GetBlockTotalDifficulty(head)
	if err != nil {
		sp.log.Error("broadcastChainHead GetBlockTotalDifficulty err. %s", err)
		return
	}

	status := &chainHeadStatus{
		TD:           localTD,
		CurrentBlock: head,
	}
	sp.peerSet.ForEach(common.LocalShardNumber, func(peer *peer) bool {
		err := peer.sendHeadStatus(status)
		if err != nil {
			sp.log.Warn("failed to send chain head info %s", err)
		}
		return true
	})
}

// syncTransactions sends pending transactions to remote peer.
func (sp *SeeleProtocol) syncTransactions(p *peer) {
	defer sp.wg.Done()
	sp.wg.Add(1)
	pending := sp.txPool.GetTransactions(false, true)

	sp.log.Debug("syncTransactions peerid:%s pending length:%d", p.peerStrID, len(pending))
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

	send(curPos)
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
	if common.PrintExplosionLog {
		p.log.Debug("find new tx")
	}

	tx := e.(*types.Transaction)

	// find shardId by tx from address.
	shardId := tx.Data.From.Shard()
	p.peerSet.ForEach(shardId, func(peer *peer) bool {
		if err := peer.sendTransactionHash(tx.Hash); err != nil {
			p.log.Warn("failed to send transaction to %s, %s", peer.Node.GetUDPAddr(), err)
		}
		return true
	})
}

func (p *SeeleProtocol) handleNewMinedBlock(e event.Event) {
	block := e.(*types.Block)

	p.peerSet.ForEach(common.LocalShardNumber, func(peer *peer) bool {
		err := peer.SendBlockHash(block.HeaderHash)
		if err != nil {
			p.log.Warn("failed to send mined block hash %s", err.Error())
		}
		return true
	})

	p.log.Debug("handleNewMinedBlock broadcast chainhead changed. new block: %d %s <- %s ",
		block.Header.Height, block.HeaderHash.ToHex(), block.Header.PreviousBlockHash.ToHex())

	p.broadcastChainHead()
}

func (p *SeeleProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) {
	if p.peerSet.Find(p2pPeer.Node.ID) != nil {
		p.log.Error("handleAddPeer called, but peer of this public-key has already existed, so need quit!")
		return
	}

	newPeer := newPeer(SeeleVersion, p2pPeer, rw, p.log)

	block := p.chain.CurrentBlock()
	head := block.HeaderHash
	localTD, err := p.chain.GetStore().GetBlockTotalDifficulty(head)
	if err != nil {
		return
	}

	if err := newPeer.handShake(p.networkID, localTD, head, common.EmptyHash); err != nil {
		p.log.Error("handleAddPeer err. %s", err)
		newPeer.Disconnect(DiscHandShakeErr)
		return
	}

	p.log.Info("add peer %s -> %s to SeeleProtocol.", p2pPeer.LocalAddr(), p2pPeer.RemoteAddr())
	p.peerSet.Add(newPeer)
	p.downloader.RegisterPeer(newPeer.peerStrID, newPeer)
	go p.syncTransactions(newPeer)
	go p.handleMsg(newPeer)
}

func (s *SeeleProtocol) handleGetPeer(address common.Address) interface{} {
	if p := s.peerSet.peerMap[address]; p != nil {
		return p.Info()
	}
	return nil
}

func (s *SeeleProtocol) handleDelPeer(peer *p2p.Peer) {
	s.log.Debug("delete peer from peer set. %s", peer.Node)
	s.peerSet.Remove(peer.Node.ID)
	s.downloader.UnRegisterPeer(idToStr(peer.Node.ID))
}

func (p *SeeleProtocol) SendDifferentShardTx(tx *types.Transaction, shard uint) {
	sendTxFun := func(peer *peer) bool {
		if !peer.knownTxs.Contains(tx.Hash) {
			err := peer.sendTransaction(tx)
			if err != nil {
				p.log.Warn("failed to send transaction to peer %s, tx hash %s", peer.Node, tx.Hash)
				return true
			}

			peer.knownTxs.Add(tx.Hash, nil)
		}

		return true
	}

	if p.peerSet.getPeerCountByShard(shard) > 0 {
		p.peerSet.ForEach(shard, sendTxFun)
	} else {
		p.peerSet.ForEachAll(sendTxFun)
	}
}

func (p *SeeleProtocol) handleMsg(peer *peer) {
handler:
	for {
		msg, err := peer.rw.ReadMsg()
		if err != nil {
			p.log.Error("get error when read msg from %s, %s", peer.peerStrID, err)
			break
		}

		// skip unsupported message from different shard peer
		if peer.Node.Shard != common.LocalShardNumber {
			if msg.Code != transactionsMsgCode {
				continue
			}
		}

		if common.PrintExplosionLog {
			p.log.Debug("got msg with type:%s", codeToStr(msg.Code))
		}

		switch msg.Code {
		case transactionHashMsgCode:
			var txHash common.Hash
			err := common.Deserialize(msg.Payload, &txHash)
			if err != nil {
				p.log.Warn("failed to deserialize transaction hash msg, %s", err.Error())
				continue
			}

			if common.PrintExplosionLog {
				p.log.Debug("got tx hash %s", txHash.ToHex())
			}

			if !peer.knownTxs.Contains(txHash) {
				peer.knownTxs.Add(txHash, nil) //update peer known transaction
				err := peer.sendTransactionRequest(txHash)
				if err != nil {
					p.log.Warn("failed to send transaction request msg, %s", err.Error())
					break handler
				}
			} else {
				if common.PrintExplosionLog {
					p.log.Debug("already have this tx %s", txHash.ToHex())
				}
			}

		case transactionRequestMsgCode:
			var txHash common.Hash
			err := common.Deserialize(msg.Payload, &txHash)
			if err != nil {
				p.log.Warn("failed to deserialize transaction request msg %s", err.Error())
				continue
			}

			if common.PrintExplosionLog {
				p.log.Debug("got tx request %s", txHash.ToHex())
			}

			tx := p.txPool.GetTransaction(txHash)
			if tx == nil {
				p.log.Debug("[transactionRequestMsgCode] not found tx in tx pool %s", txHash.ToHex())
				continue
			}

			err = peer.sendTransaction(tx)
			if err != nil {
				p.log.Warn("failed to send transaction msg %s", err.Error())
				break handler
			}

		case transactionsMsgCode:
			var txs []*types.Transaction
			err := common.Deserialize(msg.Payload, &txs)
			if err != nil {
				p.log.Warn("failed to deserialize transaction msg %s", err.Error())
				break
			}

			if common.PrintExplosionLog {
				p.log.Debug("received %d transactions", len(txs))
			}

			for _, tx := range txs {
				peer.knownTxs.Add(tx.Hash, nil)
				shard := tx.Data.From.Shard()
				if shard != common.LocalShardNumber {
					p.SendDifferentShardTx(tx, shard)
					continue
				} else {
					p.txPool.AddTransaction(tx)
				}
			}

		case blockHashMsgCode:
			var blockHash common.Hash
			err := common.Deserialize(msg.Payload, &blockHash)
			if err != nil {
				p.log.Warn("failed to deserialize block hash msg %s", err.Error())
				continue
			}

			p.log.Debug("got block hash msg %s", blockHash.ToHex())

			if !peer.knownBlocks.Contains(blockHash) {
				peer.knownBlocks.Add(blockHash, nil)
				err := peer.SendBlockRequest(blockHash)
				if err != nil {
					p.log.Warn("failed to send block request msg %s", err.Error())
					break handler
				}
			}

		case blockRequestMsgCode:
			var blockHash common.Hash
			err := common.Deserialize(msg.Payload, &blockHash)
			if err != nil {
				p.log.Warn("failed to deserialize block request msg %s", err.Error())
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
				p.log.Warn("failed to send block msg %s", err.Error())
			}

		case blockMsgCode:
			var block types.Block
			err := common.Deserialize(msg.Payload, &block)
			if err != nil {
				p.log.Warn("failed to deserialize block msg %s", err.Error())
				continue
			}

			p.log.Info("got block message and save it. height:%d, hash:%s", block.Header.Height, block.HeaderHash.ToHex())
			peer.knownBlocks.Add(block.HeaderHash, nil)
			if block.GetShardNumber() == common.LocalShardNumber {
				// @todo need to make sure WriteBlock handle block fork
				p.chain.WriteBlock(&block)
			}

		case downloader.GetBlockHeadersMsg:
			var query blockHeadersQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				p.log.Error("failed to deserialize downloader.GetBlockHeadersMsg, quit! %s", err.Error())
				break
			}
			var headList []*types.BlockHeader
			var head *types.BlockHeader
			orgNum := query.Number

			if query.Hash != common.EmptyHash {
				if head, err = p.chain.GetStore().GetBlockHeader(query.Hash); err != nil {
					p.log.Error("HandleMsg GetBlockHeader err from query hash. %s", err)
					break
				}
				orgNum = head.Height
			}

			p.log.Debug("Received downloader.GetBlockHeadersMsg start %d, amount %d", orgNum, query.Amount)
			maxHeight := p.chain.CurrentBlock().Header.Height
			for cnt := uint64(0); cnt < query.Amount; cnt++ {
				var curNum uint64
				if query.Reverse {
					curNum = orgNum - cnt
				} else {
					curNum = orgNum + cnt
				}

				if curNum > maxHeight {
					break
				}
				hash, err := p.chain.GetStore().GetBlockHash(curNum)
				if err != nil {
					p.log.Error("get error when get block hash by height. err=%s curNum=%d", err, curNum)
					break
				}

				if head, err = p.chain.GetStore().GetBlockHeader(hash); err != nil {
					p.log.Error("get error when get block by block hash. err: %s, hash:%s", err, hash)
					break
				}
				headList = append(headList, head)
			}

			if err = peer.sendBlockHeaders(query.Magic, headList); err != nil {
				p.log.Error("HandleMsg sendBlockHeaders err. %s", err)
				break handler
			}
			p.log.Debug("send downloader.sendBlockHeaders. len=%d", len(headList))

		case downloader.GetBlocksMsg:
			p.log.Debug("Received downloader.GetBlocksMsg")
			var query blocksQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				p.log.Error("failed to deserialize downloader.GetBlocksMsg, quit! %s", err.Error())
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

			p.log.Debug("Received downloader.GetBlocksMsg length %d, start %d, end %d", query.Amount, orgNum, orgNum+query.Amount)

			totalLen := 0
			var numL []uint64
			for cnt := uint64(0); cnt < query.Amount; cnt++ {
				curNum := orgNum + cnt
				hash, err := p.chain.GetStore().GetBlockHash(curNum)
				if err != nil {
					p.log.Warn("failed to get block with height %d, err %s", curNum, err)
					break
				}

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

			if len(blocksL) == 0 {
				p.log.Debug("send blocks with empty")
			} else {
				p.log.Debug("send blocks length %d, start %d, end %d", len(blocksL), blocksL[0].Header.Height, blocksL[len(blocksL)-1].Header.Height)
			}

			if err = peer.sendBlocks(query.Magic, blocksL); err != nil {
				p.log.Error("HandleMsg GetBlocksMsg sendBlocks err. %s", err)
				break handler
			}

			p.log.Debug("send downloader.sendBlocks")

		case downloader.BlockHeadersMsg, downloader.BlocksPreMsg, downloader.BlocksMsg:
			p.log.Debug("Received downloader Msg. %s peerid:%s", codeToStr(msg.Code), peer.peerStrID)
			p.downloader.DeliverMsg(peer.peerStrID, &msg)

		case statusChainHeadMsgCode:
			var status chainHeadStatus
			err := common.Deserialize(msg.Payload, &status)
			if err != nil {
				p.log.Error("failed to deserialize statusChainHeadMsgCode, quit! %s", err.Error())
				break
			}

			p.log.Debug("Received statusChainHeadMsgCode")
			peer.SetHead(status.CurrentBlock, status.TD)
			p.syncCh <- struct{}{}

		default:
			p.log.Warn("unknown code %s", msg.Code)
		}
	}

	p.handleDelPeer(peer.Peer)
	p.log.Debug("seele.peer.run out!peer=%s!", peer.peerStrID)
}
