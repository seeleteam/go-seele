/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"
	"fmt"
	rand2 "math/rand"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

const (
	statusDataMsgCode           uint16 = 0
	announceRequestCode         uint16 = 1
	announceCode                uint16 = 2
	syncHashRequestCode         uint16 = 3
	syncHashResponseCode        uint16 = 4
	downloadHeadersRequestCode  uint16 = 5
	downloadHeadersResponseCode uint16 = 6

	msgWaitTimeout = time.Second * 120
)

var (
	errReadChain = errors.New("Load message from chain err")
)

// BlockChain define some interfaces related to underlying blockchain
type BlockChain interface {
	GetCurrentState() (*state.Statedb, error)
	GetState(root common.Hash) (*state.Statedb, error)
	GetStateByRootAndBlockHash(root, blockHash common.Hash) (*state.Statedb, error)
	GetStore() store.BlockchainStore
	CurrentHeader() *types.BlockHeader
	WriteHeader(*types.BlockHeader) error
}

// TransactionPool define some interfaces related to add and get txs
type TransactionPool interface {
	AddTransaction(tx *types.Transaction) error
	GetTransaction(txHash common.Hash) *types.Transaction
}

func codeToStr(code uint16) string {
	switch code {
	case statusDataMsgCode:
		return "statusDataMsgCode"
	case announceRequestCode:
		return "announceRequestCode"
	case announceCode:
		return "announceCode"
	case syncHashRequestCode:
		return "syncHashRequestCode"
	case syncHashResponseCode:
		return "syncHashResponseCode"
	case downloadHeadersRequestCode:
		return "downloadHeadersRequestCode"
	case downloadHeadersResponseCode:
		return "downloadHeadersResponseCode"
	case blockRequestCode:
		return "blockRequestCode"
	case blockResponseCode:
		return "blockResponseCode"
	case addTxRequestCode:
		return "addTxRequestCode"
	case addTxResponseCode:
		return "addTxResponseCode"
	case trieRequestCode:
		return "trieRequestCode"
	case trieResponseCode:
		return "trieResponseCode"
	case receiptRequestCode:
		return "receiptRequestCode"
	case receiptResponseCode:
		return "receiptResponseCode"
	case txByHashRequestCode:
		return "txByHashRequestCode"
	case txByHashResponseCode:
		return "txByHashResponseCode"
	case protocolMsgCodeLength:
		return "protocolMsgCodeLength"
	}

	return "unknown"
}

// LightProtocol service implementation of seele
type LightProtocol struct {
	p2p.Protocol

	bServerMode         bool
	networkID           string
	txPool              TransactionPool
	debtPool            *core.DebtPool
	chain               BlockChain
	peerSet             *peerSet
	odrBackend          *odrBackend
	downloader          *Downloader
	wg                  sync.WaitGroup
	quitCh              chan struct{}
	syncCh              chan struct{}
	chainHeaderChangeCh chan common.Hash
	log                 *log.SeeleLog

	shard uint
}

// NewLightProtocol create LightProtocol
func NewLightProtocol(networkID string, txPool TransactionPool, debtPool *core.DebtPool, chain BlockChain, serverMode bool, odrBackend *odrBackend,
	log *log.SeeleLog, shard uint) (s *LightProtocol, err error) {
	s = &LightProtocol{
		Protocol: p2p.Protocol{
			Name:    fmt.Sprintf("%s_%d", LightProtoName, shard),
			Version: LightSeeleVersion,
			Length:  protocolMsgCodeLength,
		},
		bServerMode: serverMode,
		networkID:   networkID,
		txPool:      txPool,
		debtPool:    debtPool,
		chain:       chain,
		log:         log,
		odrBackend:  odrBackend,
		quitCh:      make(chan struct{}),
		syncCh:      make(chan struct{}),
		peerSet:     newPeerSet(),
		shard:       shard,
	}

	if !serverMode {
		s.downloader = newDownloader(chain)
	}

	s.Protocol.AddPeer = s.handleAddPeer
	s.Protocol.DeletePeer = s.handleDelPeer
	s.Protocol.GetPeer = s.handleGetPeer
	return s, nil
}

// Start starts data syncer
func (lp *LightProtocol) Start() {
	lp.log.Debug("LightProtocol.Start called!")
	if !lp.bServerMode {
		go lp.syncer()
	}
}

// Stop stops protocol, called when seeleService quits.
func (lp *LightProtocol) Stop() {
	close(lp.quitCh)
	close(lp.syncCh)
	lp.wg.Wait()
}

// syncer try to synchronise with remote peer
func (lp *LightProtocol) syncer() {
	defer lp.downloader.Terminate()
	defer lp.wg.Done()
	lp.wg.Add(1)

	forceSync := time.NewTicker(forceSyncInterval)
	for {
		select {
		case <-lp.syncCh:
			go lp.synchronise(lp.peerSet.bestPeer())
		case <-forceSync.C:
			go lp.synchronise(lp.peerSet.bestPeer())
		case <-lp.quitCh:
			return
		}
	}
}

func (lp *LightProtocol) synchronise(p *peer) {
	if p == nil {
		return
	}

	hash, err := lp.chain.GetStore().GetHeadBlockHash()
	if err != nil {
		lp.log.Error("lp.synchronise GetHeadBlockHash err.[%s]", err)
		return
	}

	localTD, err := lp.chain.GetStore().GetBlockTotalDifficulty(hash)
	if err != nil {
		lp.log.Error("lp.synchronise GetBlockTotalDifficulty err.[%s]", err)
		return
	}
	_, pTd := p.Head()

	// if total difficulty is not smaller than remote peer td, then do not need synchronise.
	if localTD.Cmp(pTd) >= 0 {
		return
	}

	err = lp.downloader.synchronise(p)
	if err != nil {
		if err == ErrIsSynchronising {
			lp.log.Info("exit synchronise as it is already running.")
		} else {
			lp.log.Error("synchronise err. %s", err)
		}
	}
}

func (lp *LightProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) bool {
	if lp.peerSet.Find(p2pPeer.Node.ID) != nil {
		lp.log.Error("handleAddPeer called, but peer of this public-key has already existed, so need quit!")
		return false
	}

	newPeer := newPeer(LightSeeleVersion, p2pPeer, rw, lp.log, lp)
	store := lp.chain.GetStore()
	hash, err := store.GetHeadBlockHash()
	if err != nil {
		lp.log.Error("lp.handleAddPeer GetHeadBlockHash err.[%s]", err)
		return false
	}

	header, err := store.GetBlockHeader(hash)
	if err != nil {
		lp.log.Error("lp.handleAddPeer GetBlockHeader err.[%s]", err)
		return false
	}

	localTD, err := store.GetBlockTotalDifficulty(hash)
	if err != nil {
		return false
	}

	genesisBlock, err := store.GetBlockByHeight(0)
	if err != nil {
		return false
	}

	if err := newPeer.handShake(lp.networkID, localTD, hash, header.Height, genesisBlock.HeaderHash); err != nil {
		if err == errModeNotMatch {
			lp.log.Info("handleAddPeer message. %s", err)
		} else {
			lp.log.Error("handleAddPeer err. %s", err)
		}

		return false
	}

	if lp.bServerMode {
		rand2.Seed(time.Now().UnixNano())
		magic := rand2.Uint32()
		if err := newPeer.sendAnnounce(magic, 0, 0); err != nil {
			lp.log.Error("sendAnnounce err. %s", err)
			newPeer.Disconnect(DiscAnnounceErr)
			return false
		}
	}

	lp.log.Info("add peer %s -> %s to LightProtocol.", p2pPeer.LocalAddr(), p2pPeer.RemoteAddr())
	lp.peerSet.Add(newPeer)
	go lp.handleMsg(newPeer)
	return true
}

func (lp *LightProtocol) handleGetPeer(address common.Address) interface{} {
	if p := lp.peerSet.peerMap[address]; p != nil {
		return p.Info()
	}

	return nil
}

func (lp *LightProtocol) handleDelPeer(peer *p2p.Peer) {
	lp.log.Debug("delete peer from peer set. %s", peer.Node)
	if p := lp.peerSet.Find(peer.Node.ID); p != nil {
		p.close()
	}

	lp.peerSet.Remove(peer.Node.ID)
}

func (lp *LightProtocol) handleMsg(peer *peer) {
handler:
	for {
		msg, err := peer.rw.ReadMsg()
		if err != nil {
			lp.log.Error("get error when read msg from %s, %s", peer.peerStrID, err)
			break
		}

		bNeedDeliverOdr := false
		switch msg.Code {
		case announceRequestCode:
			var query AnnounceQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				lp.log.Error("failed to deserialize AnnounceQuery, quit! %s", err)
				break handler
			}

			if err := peer.sendAnnounce(query.Magic, query.Begin, query.End); err != nil {
				lp.log.Error("failed to sendAnnounce, quit! %s", err)
				break handler
			}

		case announceCode:
			var query AnnounceBody
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				lp.log.Error("failed to deserialize Announce, quit! %s", err)
				break handler
			}

			if err := peer.handleAnnounce(&query); err != nil {
				lp.log.Error("failed to handleAnnounce, quit! %s", err)
				break handler
			}

		case syncHashRequestCode:
			var query HeaderHashSyncQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				lp.log.Error("failed to deserialize HeaderHashSyncQuery, quit! %s", err)
				break handler
			}

			if err := peer.handleSyncHashRequest(&query); err != nil {
				lp.log.Error("failed to handleSyncHashRequest, quit! %s", err)
				break handler
			}

		case syncHashResponseCode:
			var query HeaderHashSync
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				lp.log.Error("failed to deserialize syncHashResponseCode, quit! %s", err)
				break handler
			}

			if err := peer.handleSyncHash(&query); err != nil {
				lp.log.Error("failed to syncHashResponseCode, quit! %s", err)
				break handler
			}

		case downloadHeadersRequestCode:
			var query DownloadHeaderQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				lp.log.Error("failed to deserialize DownloadHeaderQuery, quit! %s", err)
				break handler
			}

			if err := peer.handleDownloadHeadersRequest(&query); err != nil {
				lp.log.Error("failed to DownloadHeaderQuery, quit! %s", err)
				break handler
			}

		case downloadHeadersResponseCode:
			lp.downloader.deliverMsg(peer, msg)

		default:
			if odrResponseFactories[msg.Code] != nil {
				bNeedDeliverOdr = true
			} else if err := lp.handleOdrRequest(peer, msg); err != nil {
				lp.log.Error("Failed to handle ODR message, code = %s, error = %s", codeToStr(msg.Code), err)
				break handler
			}
		}

		if bNeedDeliverOdr {
			lp.odrBackend.msgCh <- msg
		}
	}

	lp.handleDelPeer(peer.Peer)
	lp.log.Debug("light.protocol.handlemsg run out!peer= %s!", peer.peerStrID)
	peer.Disconnect(fmt.Sprintf("called from light.protocol.handlemsg. id=%s", peer.peerStrID))
}

func (lp *LightProtocol) handleOdrRequest(peer *peer, msg *p2p.Message) error {
	factory, ok := odrRequestFactories[msg.Code]
	if !ok {
		return nil
	}

	request := factory()
	if err := common.Deserialize(msg.Payload, request); err != nil {
		return fmt.Errorf("deserialize request failed with %s", err)
	}

	lp.log.Debug("begin to handle ODR request, code = %v, payloadLen = %v", codeToStr(msg.Code), len(msg.Payload))
	respCode, response := request.handle(lp)
	buff := common.SerializePanic(response)
	lp.log.Debug("peer send response, code = %v, payloadSizeBytes = %v, peerID = %v", codeToStr(respCode), len(buff), peer.peerStrID)

	return p2p.SendMessage(peer.rw, respCode, buff)
}

// GetProtocolVersion gets protocol version
func (lp *LightProtocol) GetProtocolVersion() (uint, error) {
	return lp.Protocol.Version, nil
}

// SendDifferentShardTx send tx to another shard
func (lp *LightProtocol) SendDifferentShardTx(tx *types.Transaction, shard uint) {
	//@todo
}
