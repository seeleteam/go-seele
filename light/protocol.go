/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"
	rand2 "math/rand"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
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

	protocolMsgCodeLength uint16 = 9
	msgWaitTimeout               = time.Second * 120
)

var (
	errReadChain = errors.New("Load message from chain err")
)

type BlockChain interface {
	GetCurrentState() (*state.Statedb, error)
	GetState(root common.Hash) (*state.Statedb, error)
	GetStore() store.BlockchainStore
	CurrentHeader() *types.BlockHeader
	WriteHeader(*types.BlockHeader) error
}

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
	}

	return "unknown"
}

// SeeleProtocol service implementation of seele
type LightProtocol struct {
	p2p.Protocol

	bServerMode         bool
	networkID           uint64
	txPool              TransactionPool
	chain               BlockChain
	peerSet             *peerSet
	odrBackend          *odrBackend
	downloader          *Downloader
	wg                  sync.WaitGroup
	quitCh              chan struct{}
	syncCh              chan struct{}
	chainHeaderChangeCh chan common.Hash
	log                 *log.SeeleLog
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
	sp.log.Debug("LightProtocol.Start called!")
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

	hash, err := sp.chain.GetStore().GetHeadBlockHash()
	if err != nil {
		sp.log.Error("sp.synchronise GetHeadBlockHash err.[%s]", err)
		return
	}

	localTD, err := sp.chain.GetStore().GetBlockTotalDifficulty(hash)
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

func (sp *LightProtocol) handleAddPeer(p2pPeer *p2p.Peer, rw p2p.MsgReadWriter) bool {
	if sp.peerSet.Find(p2pPeer.Node.ID) != nil {
		sp.log.Error("handleAddPeer called, but peer of this public-key has already existed, so need quit!")
		return false
	}

	newPeer := newPeer(LightSeeleVersion, p2pPeer, rw, sp.log, sp)
	store := sp.chain.GetStore()
	hash, err := store.GetHeadBlockHash()
	if err != nil {
		sp.log.Error("sp.handleAddPeer GetHeadBlockHash err.[%s]", err)
		return false
	}

	header, err := store.GetBlockHeader(hash)
	if err != nil {
		sp.log.Error("sp.handleAddPeer GetBlockHeader err.[%s]", err)
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

	if err := newPeer.handShake(sp.networkID, localTD, hash, header.Height, genesisBlock.HeaderHash); err != nil {
		sp.log.Error("handleAddPeer err. %s", err)
		if !sp.bServerMode {
			// just quit connection.
			newPeer.Disconnect(DiscHandShakeErr)
		}
		return false
	}

	if sp.bServerMode {
		rand2.Seed(time.Now().UnixNano())
		magic := rand2.Uint32()
		if err := newPeer.sendAnnounce(magic, 0, 0); err != nil {
			sp.log.Error("sendAnnounce err. %s", err)
			newPeer.Disconnect(DiscAnnounceErr)
			return false
		}
	}

	sp.log.Info("add peer %s -> %s to LightProtocol.", p2pPeer.LocalAddr(), p2pPeer.RemoteAddr())
	sp.peerSet.Add(newPeer)
	go sp.handleMsg(newPeer)
	return true
}

func (sp *LightProtocol) handleGetPeer(address common.Address) interface{} {
	if p := sp.peerSet.peerMap[address]; p != nil {
		return p.Info()
	}

	return nil
}

func (sp *LightProtocol) handleDelPeer(peer *p2p.Peer) {
	sp.log.Debug("delete peer from peer set. %s", peer.Node)
	if p := sp.peerSet.Find(peer.Node.ID); p != nil {
		p.close()
	}

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
		case announceRequestCode:
			var query AnnounceQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				sp.log.Error("failed to deserialize AnnounceQuery, quit! %s", err)
				break handler
			}

			if err := peer.sendAnnounce(query.Magic, query.Begin, query.End); err != nil {
				sp.log.Error("failed to sendAnnounce, quit! %s", err)
				break handler
			}

		case announceCode:
			var query AnnounceBody
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
				sp.log.Error("failed to deserialize syncHashResponseCode, quit! %s", err)
				break handler
			}

			if err := peer.handleSyncHash(&query); err != nil {
				sp.log.Error("failed to syncHashResponseCode, quit! %s", err)
				break handler
			}

		case downloadHeadersRequestCode:
			var query DownloadHeaderQuery
			err := common.Deserialize(msg.Payload, &query)
			if err != nil {
				sp.log.Error("failed to deserialize DownloadHeaderQuery, quit! %s", err)
				break handler
			}

			if err := peer.handleDownloadHeadersRequest(&query); err != nil {
				sp.log.Error("failed to DownloadHeaderQuery, quit! %s", err)
				break handler
			}

		case downloadHeadersResponseCode:
			sp.downloader.deliverMsg(peer, msg)

		default:
			if odrResponseFactories[msg.Code] != nil {
				bNeedDeliverOdr = true
			} else if err := sp.handleOdrRequest(peer, msg); err != nil {
				sp.log.Error("Failed to handle ODR message, code = %v, error = %v", msg.Code, err.Error())
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

func (sp *LightProtocol) handleOdrRequest(peer *peer, msg *p2p.Message) error {
	factory, ok := odrRequestFactories[msg.Code]
	if !ok {
		return nil
	}

	request := factory()
	if err := common.Deserialize(msg.Payload, request); err != nil {
		return err
	}

	sp.log.Debug("begin to handle ODR request, code = %v, payloadLen = %v", msg.Code, len(msg.Payload))
	respCode, response := request.handleRequest(sp)
	buff := common.SerializePanic(response)
	sp.log.Debug("peer send response, code = %v, payloadSizeBytes = %v, peerID = %v", respCode, len(buff), peer.peerStrID)

	return p2p.SendMessage(peer.rw, respCode, buff)
}

func (p *LightProtocol) GetProtocolVersion() (uint, error) {
	return p.Protocol.Version, nil
}

func (p *LightProtocol) SendDifferentShardTx(tx *types.Transaction, shard uint) {
	//@todo
}
