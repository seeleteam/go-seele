/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
)

const (
	DiscHandShakeErr = "disconnect because get error when handshaking of light mode"
	DiscAnnounceErr  = "disconnect because send announce message err"
)

var (
	errMsgNotMatch     = errors.New("Message not match")
	errNetworkNotMatch = errors.New("NetworkID not match")
	errModeNotMatch    = errors.New("server/client mode not match")
	errBlockNotFound   = errors.New("block not found")
)

// PeerInfo represents a short summary of a connected peer.
type PeerInfo struct {
	Version    uint     `json:"version"`    // Seele protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // SHA3 hash of the peer's best owned block
}

type peer struct {
	*p2p.Peer
	quitCh          chan struct{}
	peerStrID       string
	peerID          common.Address
	version         uint // Seele protocol version negotiated
	head            common.Hash
	headBlockNum    uint64
	td              *big.Int // total difficulty
	lock            sync.RWMutex
	protocolManager *LightProtocol
	rw              p2p.MsgReadWriter // the read write method for this peer

	blockNumBegin uint64        // first block number of blockHashArr
	blockHashArr  []common.Hash // block hashes that should be identical with remote server peer, and is only useful in client mode.
	log           *log.SeeleLog
}

func idToStr(id common.Address) string {
	return fmt.Sprintf("%x", id[:8])
}

func newPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter, log *log.SeeleLog, protocolManager *LightProtocol) *peer {
	return &peer{
		Peer:            p,
		quitCh:          make(chan struct{}),
		version:         version,
		td:              big.NewInt(0),
		peerStrID:       idToStr(p.Node.ID),
		peerID:          p.Node.ID,
		rw:              rw,
		protocolManager: protocolManager,
		log:             log,
	}
}

func (p *peer) close() {
	if p.quitCh != nil {
		select {
		case <-p.quitCh:
		default:
			close(p.quitCh)
		}
	}
}

// isSyncing returns whether synchronization is in progress.
func (p *peer) isSyncing() bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	size := len(p.blockHashArr)
	if size == 0 {
		return true
	}

	return p.blockHashArr[size-1] != p.head
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *peer) Info() *PeerInfo {
	hash, td := p.Head()

	return &PeerInfo{
		Version:    p.version,
		Difficulty: td,
		Head:       hex.EncodeToString(hash[0:]),
	}
}

// Head retrieves a copy of the current head hash and total difficulty.
func (p *peer) Head() (hash common.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.td)
}

func (p *peer) findAncestor() (uint64, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.blockHashArr) == 0 {
		return 0, errBlockNotFound
	}

	chain := p.protocolManager.chain
	for idx := len(p.blockHashArr) - 1; idx >= 0; idx-- {
		curNum, curHash := p.blockNumBegin+uint64(idx), p.blockHashArr[idx]
		localBlock, err := chain.GetStore().GetBlockByHeight(curNum)
		if err != nil {
			continue
		}

		if localBlock.HeaderHash == curHash {
			return curNum, nil
		}
	}

	return 0, errBlockNotFound
}

// RequestBlocksByHashOrNumber fetches a block according to hash or block number.
func (p *peer) RequestBlocksByHashOrNumber(reqID uint32, origin common.Hash, num uint64) error {
	query := &blockQuery{
		ReqID:  reqID,
		Hash:   origin,
		Number: num,
	}

	buff := common.SerializePanic(query)
	p.log.Debug("peer send [blockRequestMsgCode] query with size %d byte", len(buff))
	return p2p.SendMessage(p.rw, blockRequestMsgCode, buff)
}

func (p *peer) sendDownloadHeadersRequest(reqID uint32, begin uint64) error {
	query := &DownloadHeaderQuery{
		ReqID:    reqID,
		BeginNum: begin,
	}

	buff := common.SerializePanic(query)
	p.log.Debug("peer send [downloadHeadersRequestCode] query with size %d byte", len(buff))
	return p2p.SendMessage(p.rw, downloadHeadersRequestCode, buff)
}

func (p *peer) handleDownloadHeadersRequest(msg *DownloadHeaderQuery) error {
	chain := p.protocolManager.chain
	var headers []*types.BlockHeader
	beginNum := msg.BeginNum
	for i := uint64(0); i < MaxBlockHeaderRequest; i++ {
		if block, err := chain.GetStore().GetBlockByHeight(beginNum + i); err == nil {
			headers = append(headers, block.Header)
			continue
		}
		break
	}

	sendMsg := &DownloadHeader{
		ReqID:       msg.ReqID,
		HasFinished: false,
		Hearders:    headers,
	}

	if len(headers) > 0 && headers[len(headers)-1].Hash() == chain.CurrentBlock().HeaderHash {
		sendMsg.HasFinished = true
	}

	buff := common.SerializePanic(sendMsg)
	p.log.Debug("peer send [downloadHeadersResponseCode] query with size %d byte", len(buff))
	return p2p.SendMessage(p.rw, downloadHeadersResponseCode, buff)
}

func (p *peer) sendBlock(reqID uint32, block *types.Block) error {
	sendMsg := &BlockMsgBody{
		ReqID: reqID,
		Block: block,
	}
	buff := common.SerializePanic(sendMsg)

	p.log.Debug("peer send [blockMsgCode] with length: size:%d byte peerid:%s", len(buff), p.peerStrID)
	return p2p.SendMessage(p.rw, blockMsgCode, buff)
}

func (p *peer) sendSyncHashRequest(begin uint64) error {
	sendMsg := &HeaderHashSyncQuery{
		BeginNum: begin,
	}
	buff := common.SerializePanic(sendMsg)

	p.log.Debug("peer send [syncHashRequestCode] with length: size:%d byte peerid:%s", len(buff), p.peerStrID)
	return p2p.SendMessage(p.rw, syncHashRequestCode, buff)
}

// handleSyncHashRequest reponses syncHashRequestCode request, this should only be called by server mode.
func (p *peer) handleSyncHashRequest(msg *HeaderHashSyncQuery) error {
	chain := p.protocolManager.chain
	head := chain.CurrentBlock()
	localTD, err := chain.GetStore().GetBlockTotalDifficulty(head.HeaderHash)
	if err != nil {
		return errReadChain
	}

	height := head.Header.Height
	syncMsg := &HeaderHashSync{
		TD:              localTD,
		CurrentBlock:    head.HeaderHash,
		CurrentBlockNum: height,
		BeginNum:        msg.BeginNum,
	}

	var headerArr []common.Hash
	if height >= msg.BeginNum {
		count := height - msg.BeginNum + 1
		if count > MaxBlockHashRequest {
			count = MaxBlockHashRequest
		}

		for i := uint64(0); i < count; i++ {
			num := msg.BeginNum + i
			block, err := chain.GetStore().GetBlockByHeight(num)
			if err != nil {
				break
			}
			headerArr = append(headerArr, block.HeaderHash)
		}
	}
	syncMsg.HeaderArr = headerArr
	buff := common.SerializePanic(syncMsg)

	p.log.Debug("peer send [syncHashResponseCode] with length: size:%d byte peerid:%s", len(buff), p.peerStrID)
	return p2p.SendMessage(p.rw, syncHashResponseCode, buff)
}

// findIdxByHash finds index of hash in p.blockHashArr, and returns -1 if not found
func (p *peer) findIdxByHash(hash common.Hash) int {
	if len(p.blockHashArr) == 0 {
		return -1
	}

	for idx := 0; idx < len(p.blockHashArr); idx++ {
		if p.blockHashArr[idx] == hash {
			return idx
		}
	}

	return -1
}

// handleSyncHash handles HeaderHashSync message, this should only be called by client mode
func (p *peer) handleSyncHash(msg *HeaderHashSync) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.td, p.head, p.headBlockNum = msg.TD, msg.CurrentBlock, msg.CurrentBlockNum
	if len(msg.HeaderArr) <= 1 {
		return nil
	}

	if len(p.blockHashArr) == 0 {
		p.blockNumBegin, p.blockHashArr = msg.BeginNum, msg.HeaderArr
		return nil
	}

	idx := p.findIdxByHash(p.blockHashArr[0])
	if idx < 0 {
		p.log.Info("handleSyncHash hash not match")
		return nil
	}

	p.blockHashArr = append(p.blockHashArr[0:idx], msg.HeaderArr...)
	return nil
}

// sendAnnounce sends header hash between [begin,end] selectively,
// if end equals 0, end should be maximum block number in blockchain.
func (p *peer) sendAnnounce(begin uint64, end uint64) error {
	chain := p.protocolManager.chain
	if end == 0 {
		end = chain.CurrentBlock().Header.Height
	}
	//todo
	return nil
}

func (p *peer) handleAnnounce(msg *Announce) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.td, p.head, p.headBlockNum = msg.TD, msg.CurrentBlock, msg.CurrentBlockNum

	startNum := uint64(0)
	if len(p.blockHashArr) == 0 {
		// todo find common ancestor with local chain, and send AnnounceQuery if gap is big enough
		//
	} else {
		// todo find common ancestor with peer.blockHashArr
	}

	return p.sendSyncHashRequest(startNum)
}

// handShake exchange networkid td etc between two connected peers.
func (p *peer) handShake(networkID uint64, td *big.Int, head common.Hash, headBlockNum uint64, genesis common.Hash) error {
	msg := &statusData{
		ProtocolVersion: uint32(LightSeeleVersion),
		NetworkID:       networkID,
		IsServer:        p.protocolManager.bServerMode,
		TD:              td,
		CurrentBlock:    head,
		CurrentBlockNum: headBlockNum,
		GenesisBlock:    genesis,
	}

	if err := p2p.SendMessage(p.rw, statusDataMsgCode, common.SerializePanic(msg)); err != nil {
		return err
	}

	retMsg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if retMsg.Code != statusDataMsgCode {
		return errMsgNotMatch
	}

	var retStatusMsg statusData
	if err = common.Deserialize(retMsg.Payload, &retStatusMsg); err != nil {
		return err
	}

	if retStatusMsg.NetworkID != networkID || retStatusMsg.GenesisBlock != genesis {
		return errNetworkNotMatch
	}

	if retStatusMsg.IsServer == p.protocolManager.bServerMode {
		return errModeNotMatch
	}

	p.head, p.td, p.headBlockNum = retStatusMsg.CurrentBlock, retStatusMsg.TD, retStatusMsg.CurrentBlockNum
	return nil
}
