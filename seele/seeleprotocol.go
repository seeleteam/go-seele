/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common/memory"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	downloader "github.com/seeleteam/go-seele/seele/download"
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

	debtMsgCode uint16 = 13

	protocolMsgCodeLength uint16 = 14
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
	case debtMsgCode:
		return "debtMsgCode"
	}

	return downloader.CodeToStr(code)
}

// SeeleProtocol service implementation of seele
type SeeleProtocol struct {
	p2p.Protocol
	peerSet *peerSet

	networkID  string
	downloader *downloader.Downloader
	txPool     *core.TransactionPool
	debtPool   *core.DebtPool
	chain      *core.Blockchain

	wg     sync.WaitGroup
	quitCh chan struct{}
	syncCh chan struct{}
	log    *log.SeeleLog

	debtManager *DebtManager
}

// Downloader return a pointer of the downloader
func (s *SeeleProtocol) Downloader() *downloader.Downloader { return s.downloader }

// NewSeeleProtocol create SeeleProtocol
func NewSeeleProtocol(seele *SeeleService, log *log.SeeleLog) (s *SeeleProtocol, err error) {
	s = &SeeleProtocol{
		Protocol: p2p.Protocol{
			Name:    common.SeeleProtoName,
			Version: common.SeeleVersion,
			Length:  protocolMsgCodeLength,
		},
		networkID:  seele.networkID,
		txPool:     seele.TxPool(),
		debtPool:   seele.debtPool,
		chain:      seele.BlockChain(),
		downloader: downloader.NewDownloader(seele.BlockChain(), seele),
		log:        log,
		quitCh:     make(chan struct{}),
		syncCh:     make(chan struct{}),

		peerSet: newPeerSet(),
	}

	s.Protocol.AddPeer = s.handleAddPeer
	s.Protocol.DeletePeer = s.handleDelPeer
	s.Protocol.GetPeer = s.handleGetPeer

	s.debtManager = NewDebtManager(seele.debtVerifier, s, s.chain, seele.debtManagerDB)

	event.TransactionInsertedEventManager.AddAsyncListener(s.handleNewTx)
	event.BlockMinedEventManager.AddAsyncListener(s.handleNewMinedBlock)
	event.ChainHeaderChangedEventMananger.AddAsyncListener(s.handleNewBlock)
	event.DebtsInsertedEventManager.AddAsyncListener(s.handleNewDebt)
	return s, nil
}

func (sp *SeeleProtocol) Start() {
	sp.log.Debug("SeeleProtocol.Start called!")
	go sp.syncer()
	go sp.debtManager.TimingChecking()
}

// Stop stops protocol, called when seeleService quits.
func (sp *SeeleProtocol) Stop() {
	event.BlockMinedEventManager.RemoveListener(sp.handleNewMinedBlock)
	event.TransactionInsertedEventManager.RemoveListener(sp.handleNewTx)
	event.DebtsInsertedEventManager.RemoveListener(sp.handleNewDebt)
	close(sp.quitCh)
	close(sp.syncCh)
	sp.wg.Wait()
}

// syncer try to synchronise with remote peer
func (sp *SeeleProtocol) syncer() {
	defer sp.downloader.Terminate()
	//defer sp.wg.Done()
	//sp.wg.Add(1)

	forceSync := time.NewTicker(forceSyncInterval)
	for {
		select {
		case <-sp.syncCh:
			if !sp.downloader.IsSyncStatusNone() {
				continue
			}
			block := sp.chain.CurrentBlock()
			head := block.HeaderHash
			localTD, err := sp.chain.GetStore().GetBlockTotalDifficulty(head)
			if err != nil {
				sp.log.Error("broadcastChainHead GetBlockTotalDifficulty err. %s", err)
				continue
			}
			sp.wg.Add(1)
			go sp.synchronise(sp.peerSet.bestPeers(common.LocalShardNumber, localTD))
		case <-forceSync.C:
			if !sp.downloader.IsSyncStatusNone() {
				continue
			}
			block := sp.chain.CurrentBlock()
			head := block.HeaderHash
			localTD, err := sp.chain.GetStore().GetBlockTotalDifficulty(head)
			if err != nil {
				sp.log.Error("broadcastChainHead GetBlockTotalDifficulty err. %s", err)
				continue
			}
			sp.wg.Add(1)
			go sp.synchronise(sp.peerSet.bestPeers(common.LocalShardNumber, localTD))
		case <-sp.quitCh:
			return
		}
	}
}

func (sp *SeeleProtocol) synchronise(peers []*peer) {
	defer sp.wg.Done()
	now := time.Now()
	// entrance
	memory.Print(sp.log, "SeeleProtocol synchronise entrance", now, false)

	if len(peers) == 0 {
		return
	}

	//block := sp.chain.CurrentBlock()
	//localTD, err := sp.chain.GetStore().GetBlockTotalDifficulty(block.HeaderHash)
	//if err != nil {
	//	sp.log.Error("sp.synchronise GetBlockTotalDifficulty err.[%s], Hash: %v", err, block.HeaderHash)
	//	// one step
	//	memory.Print(sp.log, "SeeleProtocol synchronise GetBlockTotalDifficulty error", now, true)
	//	return
	//}

	for _, p := range peers {
		pHead, _ := p.Head()

		// if total difficulty is not smaller than remote peer td, then do not need synchronise.
		//if localTD.Cmp(pTd) >= 0 {
		// two step
		//	memory.Print(sp.log, "SeeleProtocol synchronise difficulty is bigger than remote", now, true)
		//	return //no need to continue because peers are selected to be the best peers
		//}
		err := sp.downloader.Synchronise(p.peerStrID, pHead)
		if err != nil {
			if err == downloader.ErrIsSynchronising {
				sp.log.Debug("exit synchronise as it is already running.")
			} else {
				sp.log.Debug("synchronise err. %s", err)
			}

			// three step
			memory.Print(sp.log, "SeeleProtocol synchronise downloader error", now, true)

			continue
		}

		//broadcast chain head
		sp.broadcastChainHead()

		// exit
		memory.Print(sp.log, "SeeleProtocol synchronise exit", now, true)

		return
	}
}

func (sp *SeeleProtocol) broadcastChainHead() {

	now := time.Now()
	// entrance
	memory.Print(sp.log, "SeeleProtocol broadcastChainHead entrance", now, false)

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

	peers := sp.peerSet.getAllPeers()

	wg := new(sync.WaitGroup)

	for _, peer := range peers {
		if peer != nil {
			//err := peer.sendHeadStatus(status)
			wg.Add(1)
			go peer.sendHeadStatus(status, wg)
			//if err != nil {
			//	sp.log.Warn("failed to send chain head info err=%s, id=%s, ip=%s", err, peer.peerStrID, peer.Peer.RemoteAddr())
			//} else {
			//	sp.log.Debug("send chain head info err=%s, id=%s, ip=%s, localTD=%d", err, peer.peerStrID, peer.Peer.RemoteAddr(), localTD)
			//}
		}
	}
	wg.Wait()
	// exit
	memory.Print(sp.log, "SeeleProtocol broadcastChainHead exit", now, true)
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
	now := time.Now()
	// entrance
	memory.Print(p.log, "SeeleProtocol handleNewTx entrance", now, false)

	tx := e.(*types.Transaction)

	// find shardId by tx from address.
	shardId := tx.Data.From.Shard()
	peers := p.peerSet.getPeerByShard(shardId)
	for _, peer := range peers {
		if peer.knownTxs.Contains(tx.Hash) {
			p.log.Debug("seeleprotocol handleNewTx: peer: %s already contains tx %s", peer.peerStrID, tx.Hash.String())
			continue
		}

		if err := peer.sendTransaction(tx); err != nil {
			p.log.Warn("failed to send transaction to peer=%s, err=%s", peer.Node.GetUDPAddr(), err)
			peer.Disconnect(err.Error())
		}
	}

	//exit
	memory.Print(p.log, "SeeleProtocol handleNewTx exit", now, true)
}

func (p *SeeleProtocol) handleNewDebt(e event.Event) {
	debt := e.(*types.Debt)
	p.propagateDebtMap(types.DebtArrayToMap([]*types.Debt{debt}), true)
}

func (p *SeeleProtocol) propagateDebtMap(debtsMap [][]*types.Debt, filter bool) {
	now := time.Now()
	// entrance
	memory.Print(p.log, "SeeleProtocol propagateDebtMap entrance", now, false)

	//peers := p.peerSet.getAllPeers()
	wg := new(sync.WaitGroup)
	peers := p.peerSet.getPropagatePeers()
	for _, peer := range peers {
		if len(debtsMap[peer.Node.Shard]) > 0 {
			wg.Add(1)
			go peer.sendDebts(debtsMap[peer.Node.Shard], filter)
			//err := peer.sendDebts(debtsMap[peer.Node.Shard], filter)
			//if err != nil {
			//	p.log.Warn("failed to send debts to peer=%s, err=%s", peer.Node, err)
			//}
		}
	}
	wg.Wait()
	// exit
	memory.Print(p.log, "SeeleProtocol propagateDebtMap exit", now, true)
}

func (p *SeeleProtocol) handleNewBlock(e event.Event) {
	block := e.(*types.Block)

	// propagate confirmed block
	if block.Header.Height > common.ConfirmedBlockNumber {
		confirmedHeight := block.Header.Height - common.ConfirmedBlockNumber
		confirmedBlock, err := p.chain.GetStore().GetBlockByHeight(confirmedHeight)

		if err != nil {
			p.log.Warn("failed to load confirmed block height %d, err %s", confirmedHeight, err)
		} else {
			now := time.Now()
			// entrance
			memory.Print(p.log, "SeeleProtocol handleNewBlock entrance", now, false)

			debts := types.NewDebtMap(confirmedBlock.Transactions)
			p.debtManager.AddDebtMap(debts, confirmedHeight)
			go p.propagateDebtMap(debts, true)

			// exit
			memory.Print(p.log, "SeeleProtocol handleNewBlock exit", now, true)
		}
	}
}

func (p *SeeleProtocol) handleNewMinedBlock(e event.Event) {
	now := time.Now()
	// entrance
	memory.Print(p.log, "SeeleProtocol handleNewMinedBlock entrance", now, false)
	block := e.(*types.Block)

	p.log.Debug("handleNewMinedBlock broadcast chainhead changed. new block: %d %s <- %s ",
		block.Header.Height, block.HeaderHash.Hex(), block.Header.PreviousBlockHash.Hex())

	p.broadcastChainHead()

	// exit
	memory.Print(p.log, "SeeleProtocol handleNewMinedBlock exit", now, true)
}

func (p *SeeleProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) bool {
	if p.peerSet.Find(p2pPeer.Node.ID) != nil {
		p.log.Error("handleAddPeer called, but peer of this public-key has already existed, so need quit!")
		return false
	}

	newPeer := newPeer(common.SeeleVersion, p2pPeer, rw, p.log)

	block := p.chain.CurrentBlock()
	head := block.HeaderHash
	localTD, err := p.chain.GetStore().GetBlockTotalDifficulty(head)
	if err != nil {
		return false
	}

	genesisBlock, err := p.chain.GetStore().GetBlockByHeight(0)
	if err != nil {
		return false
	}

	if err := newPeer.handShake(p.networkID, localTD, head, genesisBlock.HeaderHash, genesisBlock.Header.Difficulty.Uint64()); err != nil {
		p.log.Debug("handleAddPeer err. %s", err)
		newPeer.Disconnect(DiscHandShakeErr)
		return false
	}

	p.log.Debug("add peer %s -> %s to SeeleProtocol. nodeid=%s", p2pPeer.LocalAddr(), p2pPeer.RemoteAddr(), newPeer.peerStrID)
	p.peerSet.Add(newPeer)
	if newPeer.Node.Shard == common.LocalShardNumber {
		p.downloader.RegisterPeer(newPeer.peerStrID, newPeer)

	}
	//go p.syncTransactions(newPeer)
	go p.handleMsg(newPeer)
	return true
}

func (s *SeeleProtocol) handleGetPeer(address common.Address) interface{} {
	if p := s.peerSet.Find(address); p != nil {
		return p.Info()
	}
	return nil
}

func (s *SeeleProtocol) handleDelPeer(peer *p2p.Peer) {
	s.log.Debug("delete peer from peer set. %s", peer.Node)
	s.peerSet.Remove(peer.Node.ID)

	if peer.Node.Shard == common.LocalShardNumber {
		s.downloader.UnRegisterPeer(idToStr(peer.Node.ID))
	}
}

// SendDifferentShardTx send tx to different shards
func (p *SeeleProtocol) SendDifferentShardTx(tx *types.Transaction, shard uint) {
	var peers []*peer

	peers = p.peerSet.getPeerByShard(shard)
	if len(peers) <= 0 {
		peers = p.peerSet.getAllPeers()
	}

	for _, peerinfo := range peers {
		if !peerinfo.knownTxs.Contains(tx.Hash) {
			err := peerinfo.sendTransaction(tx)
			if err != nil {
				p.log.Warn("failed to send transaction to peer=%s, tx hash=%s, err=%s", peerinfo.Node, tx.Hash, err)
				continue
			}

			peerinfo.knownTxs.Add(tx.Hash, nil)
		}
	}
}

func (p *SeeleProtocol) handleMsg(peer *peer) {
handler:
	for {
		msg, err := peer.rw.ReadMsg()

		if err != nil {
			p.log.Debug("get error when read msg from %s, %s", peer.peerStrID, err)
			break
		}

		// skip unsupported message from different shard peer
		if peer.Node.Shard != common.LocalShardNumber {
			if msg.Code != transactionsMsgCode && msg.Code != debtMsgCode && msg.Code != statusChainHeadMsgCode {
				continue
			}
		}

		// print transaction and debt pool length
		p.log.Debug("handleMsg tx pool and debt pool length, tx %d, debt %d", p.txPool.GetTxCount(), p.debtPool.GetDebtCount(true, true))

		// set time now
		now := time.Now()

		switch msg.Code {
		case transactionHashMsgCode:
			// entrance
			memory.Print(p.log, "handleMsg transactionHashMsgCode entrance", now, false)

			var txHash common.Hash
			err := common.Deserialize(msg.Payload, &txHash)
			if err != nil {
				p.log.Warn("failed to deserialize transaction hash msg, %s", err.Error())
				continue
			}

			if !peer.knownTxs.Contains(txHash) {
				//update peer known transaction
				peer.knownTxs.Add(txHash, nil)

				if err := peer.sendTransactionRequest(txHash); err != nil {
					p.log.Warn("failed to send transaction request msg to peer=%s, err=%s", peer.RemoteAddr().String(), err.Error())
					break handler
				}

			}

			// exit
			memory.Print(p.log, "handleMsg transactionHashMsgCode exit", now, true)

		case transactionRequestMsgCode:
			// entrance
			memory.Print(p.log, "handleMsg transactionRequestMsgCode entrance", now, false)

			var txHash common.Hash
			err := common.Deserialize(msg.Payload, &txHash)
			if err != nil {
				p.log.Warn("failed to deserialize transaction request msg %s", err.Error())
				continue
			}

			tx := p.txPool.GetTransaction(txHash)
			if tx == nil {
				p.log.Debug("[transactionRequestMsgCode] not found tx in tx pool %s", txHash.Hex())
				continue
			}

			err = peer.sendTransaction(tx)
			if err != nil {
				p.log.Warn("failed to send transaction msg to peer=%s, err=%s", peer.RemoteAddr().String(), err.Error())
				break handler
			}

			// exit
			memory.Print(p.log, "handleMsg transactionRequestMsgCode exit", now, true)

		case transactionsMsgCode:
			// entrance
			memory.Print(p.log, "handleMsg transactionsMsgCode entrance", now, false)

			var txs []*types.Transaction
			err := common.Deserialize(msg.Payload, &txs)
			if err != nil {
				p.log.Warn("failed to deserialize transaction msg %s", err.Error())
				break
			}

			go func() {
				for _, tx := range txs {
					peer.knownTxs.Add(tx.Hash, nil)
					shard := tx.Data.From.Shard()
					if shard != common.LocalShardNumber {
						p.SendDifferentShardTx(tx, shard)
						continue
					} else {
						p.txPool.AddTransaction(tx)
						//if err := p.txPool.AddTransaction(tx); err != nil {
						//	continue
						//	}
					}
				}
			}()

			memory.Print(p.log, "handleMsg transactionsMsgCode exit", now, true)

		case blockHashMsgCode:
			// entrance
			memory.Print(p.log, "handleMsg blockHashMsgCode entrance", now, false)

			var blockHash common.Hash
			err := common.Deserialize(msg.Payload, &blockHash)
			if err != nil {
				p.log.Warn("failed to deserialize block hash msg %s", err.Error())
				continue
			}

			p.log.Debug("got block hash msg %s", blockHash.Hex())

			if !peer.knownBlocks.Contains(blockHash) {
				peer.knownBlocks.Add(blockHash, nil)

				err := peer.SendBlockRequest(blockHash)
				if err != nil {
					p.log.Warn("failed to send block request msg %s", err.Error())
					break handler
				}
			}

			//exit
			memory.Print(p.log, "handleMsg blockHashMsgCode exit", now, true)

		case blockRequestMsgCode:
			// entrance
			memory.Print(p.log, "handleMsg blockRequestMsgCode entrance", now, false)

			var blockHash common.Hash
			err := common.Deserialize(msg.Payload, &blockHash)
			if err != nil {
				p.log.Warn("failed to deserialize block request msg %s", err.Error())
				continue
			}

			p.log.Debug("got block request msg %s", blockHash.Hex())
			block, err := p.chain.GetStore().GetBlock(blockHash)
			if err != nil {
				p.log.Warn("not found request block %s", err.Error())
				continue
			}
			go peer.SendBlock(block)
			//err = peer.SendBlock(block)
			//if err != nil {
			//p.log.Warn("failed to send block msg to peer=%s, err=%s", peer.RemoteAddr().String(), err.Error())
			//}

			// exit
			memory.Print(p.log, "handleMsg blockRequestMsgCode exit", now, true)

		case blockMsgCode:
			// entrance
			memory.Print(p.log, "handleMsg blockMsgCode entrance", now, false)

			var block types.Block
			err := common.Deserialize(msg.Payload, &block)
			if err != nil {
				p.log.Warn("failed to deserialize block msg %s", err.Error())
				continue
			}

			p.log.Info("got block message and save it. height:%d, hash:%s, time: %d", block.Header.Height, block.HeaderHash.Hex(), time.Now().UnixNano())
			peer.knownBlocks.Add(block.HeaderHash, nil)
			if block.GetShardNumber() == common.LocalShardNumber {
				// @todo need to make sure WriteBlock handle block fork
				go p.chain.WriteBlock(&block, p.txPool.Pool)
			}

			// exit
			memory.Print(p.log, "handleMsg blockMsgCode exit", now, true)

		case debtMsgCode:
			// entrance
			memory.Print(p.log, "handleMsg debtMsgCode entrance", now, false)

			var debts []*types.Debt
			err := common.Deserialize(msg.Payload, &debts)
			if err != nil {
				p.log.Warn("failed to deserialize debts msg %s", err)
				continue
			}

			p.log.Debug("got %d debts message [%s]", len(debts), codeToStr(msg.Code))
			for _, d := range debts {
				peer.knownDebts.Add(d.Hash, nil)
			}

			go p.debtPool.AddDebtArray(debts)

			//exit
			memory.Print(p.log, "handleMsg debtMsgCode exit", now, true)

		case downloader.GetBlockHeadersMsg:
			//entrance
			memory.Print(p.log, "handleMsg downloader.GetBlockHeadersMsg entrance", now, false)

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
					p.log.Debug("HandleMsg GetBlockHeader err from query hash.err= %s magic= %d id= %d ip= %s", err, query.Magic, peer.peerID, peer.Peer.RemoteAddr())
					break
				}
				orgNum = head.Height
			}

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
					p.log.Error("get error when get block hash by height. err= %s curNum= %d magic= %d id= %s ip= %s", err, curNum, query.Magic, peer.peerID, peer.Peer.RemoteAddr())
					break
				}

				if head, err = p.chain.GetStore().GetBlockHeader(hash); err != nil {
					p.log.Error("get error when get block by block hash. err: %s, hash:%s magic=%d id=%s ip=%s", err, hash, query.Magic, peer.peerID, peer.Peer.RemoteAddr())
					break
				}
				headList = append(headList, head)
			}

			go peer.sendBlockHeaders(query.Magic, headList)

			// exit
			memory.Print(p.log, "handleMsg downloader.GetBlockHeadersMsg exit", now, true)

		case downloader.GetBlocksMsg:
			// entrance
			memory.Print(p.log, "handleMsg downloader.GetBlocksMsg entrance", now, false)

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

			p.log.Debug("Received downloader.GetBlocksMsg length %d, start %d, end %d magic= %d id= %s ip= %s", query.Amount, orgNum, orgNum+query.Amount, query.Magic, peer.peerStrID, peer.Peer.RemoteAddr())

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

			go peer.sendBlocks(query.Magic, blocksL)

			// exit
			memory.Print(p.log, "handleMsg downloader.GetBlocksMsg exit", now, true)

		case downloader.BlockHeadersMsg, downloader.BlocksPreMsg, downloader.BlocksMsg:
			// entrance
			memory.Print(p.log, "handleMsg downloader.BlockHeadersMsg, downloader.BlocksPreMsg, downloader.BlocksMsg entrance", now, false)

			p.log.Debug("Received downloader Msg. %s peerid:%s", codeToStr(msg.Code), peer.peerStrID)
			go p.downloader.DeliverMsg(peer.peerStrID, msg)

			// exit
			memory.Print(p.log, "handleMsg downloader.BlockHeadersMsg, downloader.BlocksPreMsg, downloader.BlocksMsg exit", now, true)

		case statusChainHeadMsgCode:
			// entrance
			memory.Print(p.log, "handleMsg statusChainHeadMsgCode entrance", now, false)

			var status chainHeadStatus
			err := common.Deserialize(msg.Payload, &status)
			if err != nil {
				p.log.Error("failed to deserialize statusChainHeadMsgCode, quit! %s", err.Error())
				break
			}

			p.log.Debug("Received statusChainHeadMsgCode. peer=%s, ip=%s, remoteTD=%d", peer.peerStrID, peer.Peer.RemoteAddr(), status.TD)
			peer.SetHead(status.CurrentBlock, status.TD)
			p.syncCh <- struct{}{}

			// exit
			memory.Print(p.log, "handleMsg statusChainHeadMsgCode exit", now, true)

		default:
			p.log.Warn("unknown code %d", msg.Code)
		}

	}

	p.handleDelPeer(peer.Peer)
	p.log.Debug("seele.protocol.handlemsg run out! peer= %s!", peer.peerStrID)
	peer.Disconnect(fmt.Sprintf("called from seeleprotocol.handlemsg. id=%s", peer.peerStrID))
}

func (p *SeeleProtocol) GetProtocolVersion() (uint, error) {
	return p.Protocol.Version, nil
}

func (sp *SeeleProtocol) FindPeers(targets map[common.Address]bool) map[common.Address]consensus.Peer {
	m := make(map[common.Address]consensus.Peer)
	for _, p := range sp.peerSet.getPeerByShard(common.LocalShardNumber) {
		addr := p.Node.ID
		if targets[addr] {
			m[addr] = p
		}
	}

	return m
}
