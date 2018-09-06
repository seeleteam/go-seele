/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

const (
	blockRequestMsgCode  uint16 = 0
	blockMsgCode         uint16 = 1
	statusDataMsgCode    uint16 = 2
	announceRequestCode  uint16 = 3
	announceCode         uint16 = 4
	syncHashRequestCode  uint16 = 5
	syncHashResponseCode uint16 = 6

	protocolMsgCodeLength uint16 = 7
	msgWaitTimeout               = time.Second * 120
)

var (
	errReadChain = errors.New("Load message from chain err")
)

type BlockChain interface {
	CurrentBlock() *types.Block
	GetStore() store.BlockchainStore
}

type TransactionPool interface {
	//AddRemotes(txs []*types.Transaction) []error
	//Status(hashes []common.Hash) []core.TxStatus
}

func codeToStr(code uint16) string {
	switch code {
	case blockRequestMsgCode:
		return "blockRequestMsgCode"
	case blockMsgCode:
		return "blockMsgCode"
	}

	return "unknown"
}

// SeeleProtocol service implementation of seele
type LightProtocol struct {
	p2p.Protocol

	bServerMode              bool
	networkID                uint64
	txPool                   TransactionPool
	chain                    BlockChain
	peerSet                  *peerSet
	odrBackend               *odrBackend
	downloader               *Downloader
	wg                       sync.WaitGroup
	quitCh                   chan struct{}
	syncCh                   chan struct{}
	chainHeaderChangeChannel chan common.Hash
	log                      *log.SeeleLog
}

// NewLightProtocol create LightProtocol
func NewLightProtocol(networkID uint64, txPool TransactionPool, chain BlockChain, serverMode bool, odrBackend *odrBackend, log *log.SeeleLog) (s *LightProtocol, err error) {
	s = &LightProtocol{
		Protocol: p2p.Protocol{
			Name:    LightProtoName,
			Version: LightSeeleVersion,
			Length:  protocolMsgCodeLength,
		},
		bServerMode: serverMode,
		networkID:   networkID,
		txPool:      txPool,
		chain:       chain,
		log:         log,
		odrBackend:  odrBackend,
		quitCh:      make(chan struct{}),
		syncCh:      make(chan struct{}),
		peerSet:     newPeerSet(),
	}

	if !serverMode {
		s.downloader = newDownloader(chain)
	}

	s.Protocol.AddPeer = s.handleAddPeer
	s.Protocol.DeletePeer = s.handleDelPeer
	s.Protocol.GetPeer = s.handleGetPeer
	return s, nil
}

func (sp *LightProtocol) Start() {
	sp.log.Debug("SeeleProtocol.Start called!")
	if !sp.bServerMode {
		go sp.syncer()
	}
}

// Stop stops protocol, called when seeleService quits.
func (sp *LightProtocol) Stop() {
	close(sp.quitCh)
	close(sp.syncCh)
	sp.wg.Wait()
}

// syncer try to synchronise with remote peer
func (sp *LightProtocol) syncer() {
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

func (sp *LightProtocol) synchronise(p *peer) {
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
	_, pTd := p.Head()

	// if total difficulty is not smaller than remote peer td, then do not need synchronise.
	if localTD.Cmp(pTd) >= 0 {
		return
	}

	err = sp.downloader.synchronise(p)
	if err != nil {
		if err == ErrIsSynchronising {
			sp.log.Info("exit synchronise as it is already running.")
		} else {
			sp.log.Error("synchronise err. %s", err)
		}
	}
}

func (sp *LightProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) {
	if sp.peerSet.Find(p2pPeer.Node.ID) != nil {
		sp.log.Error("handleAddPeer called, but peer of this public-key has already existed, so need quit!")
		return
	}

	newPeer := newPeer(LightSeeleVersion, p2pPeer, rw, sp.log, sp)

	block := sp.chain.CurrentBlock()
	head := block.HeaderHash
	localTD, err := sp.chain.GetStore().GetBlockTotalDifficulty(head)
	if err != nil {
		return
	}

	genesisBlock, err := sp.chain.GetStore().GetBlockByHeight(0)
	if err != nil {
		return
	}

	if err := newPeer.handShake(sp.networkID, localTD, head, block.Header.Height, genesisBlock.HeaderHash); err != nil {
		sp.log.Error("handleAddPeer err. %s", err)
		if sp.bServerMode {
			// todo. light protocol need quit, but seeleprotocol can run normally.
		} else {
			// just quit connection.
			newPeer.Disconnect(DiscHandShakeErr)
		}
		return
	}

	if sp.bServerMode {
		if err := newPeer.sendAnnounce(0, 0); err != nil {
			sp.log.Error("sendAnnounce err. %s", err)
			newPeer.Disconnect(DiscAnnounceErr)
			return
		}
	}

	sp.log.Info("add peer %s -> %s to LightProtocol.", p2pPeer.LocalAddr(), p2pPeer.RemoteAddr())
	sp.peerSet.Add(newPeer)
	go sp.handleMsg(newPeer)
}

func (sp *LightProtocol) handleGetPeer(address common.Address) interface{} {
	if p := sp.peerSet.peerMap[address]; p != nil {
		return p.Info()
	}

	return nil
}

func (sp *LightProtocol) handleDelPeer(peer *p2p.Peer) {
	sp.log.Debug("delete peer from peer set. %s", peer.Node)
	sp.peerSet.Remove(peer.Node.ID)
}

func (sp *LightProtocol) handleMsg(peer *peer) {
handler:
	for {
		msg, err := peer.rw.ReadMsg()
		if err != nil {
			sp.log.Error("get error when read msg from %s, %s", peer.peerStrID, err)
			break
		}

		if common.PrintExplosionLog {
			sp.log.Debug("got msg with type:%s", codeToStr(msg.Code))
		}

		bNeedDeliverOdr := false
		switch msg.Code {
		case blockRequestMsgCode:
			var query blockQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				sp.log.Error("failed to deserialize blockRequestMsgCode, quit! %s", err.Error())
				break handler
			}

			blockHash := query.Hash
			var block *types.Block

			if query.Hash == common.EmptyHash {
				if hash, err := sp.chain.GetStore().GetBlockHash(query.Number); err != nil {
					sp.log.Warn("failed to get block with height %d, err %s", query.Number, err)
				} else {
					blockHash = hash
				}
			}

			if block, err = sp.chain.GetStore().GetBlock(blockHash); err != nil {
				sp.log.Error("HandleMsg GetBlocksMsg p.chain.GetStore().GetBlock err. %s", err)
			}

			// block can be nil if not found
			if err = peer.sendBlock(query.ReqID, block); err != nil {
				sp.log.Error("HandleMsg GetBlocksMsg sendBlocks err. %s", err)
				break handler
			}

		case blockMsgCode:
			bNeedDeliverOdr = true
			sp.log.Debug("Received Msg. %s peerid:%s", codeToStr(msg.Code), peer.peerStrID)

		case announceRequestCode:
			var query AnnounceQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				sp.log.Error("failed to deserialize AnnounceQuery, quit! %s", err)
				break handler
			}

			if err := peer.sendAnnounce(query.Begin, query.End); err != nil {
				sp.log.Error("failed to sendAnnounce, quit! %s", err)
				break handler
			}

		case announceCode:
			var query Announce
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				sp.log.Error("failed to deserialize Announce, quit! %s", err)
				break handler
			}

			if err := peer.handleAnnounce(&query); err != nil {
				sp.log.Error("failed to handleAnnounce, quit! %s", err)
				break handler
			}

		case syncHashRequestCode:
			var query HeaderHashSyncQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				sp.log.Error("failed to deserialize HeaderHashSyncQuery, quit! %s", err)
				break handler
			}

			if err := peer.handleSyncHashRequest(&query); err != nil {
				sp.log.Error("failed to handleSyncHashRequest, quit! %s", err)
				break handler
			}

		case syncHashResponseCode:
			var query HeaderHashSync
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				sp.log.Error("failed to deserialize HeaderHashSync, quit! %s", err)
				break handler
			}

			if err := peer.handleSyncHash(&query); err != nil {
				sp.log.Error("failed to handleSyncHash, quit! %s", err)
				break handler
			}
		}

		if bNeedDeliverOdr {
			sp.odrBackend.msgCh <- msg
		}
	}

	sp.handleDelPeer(peer.Peer)
	sp.log.Debug("seele.peer.run out!peer=%s!", peer.peerStrID)
}
