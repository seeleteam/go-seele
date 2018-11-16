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
	// DiscHandShakeErr disconnect due to failed to shake hands in light mode
	DiscHandShakeErr = "disconnect because get error when handshaking of light mode"

	// DiscAnnounceErr disconnect due to failed to send announce message
	DiscAnnounceErr = "disconnect because send announce message err"
)

var (
	errMsgNotMatch     = errors.New("message mismatch")
	errNetworkNotMatch = errors.New("networkID mismatch")
	errModeNotMatch    = errors.New("server/client mode mismatch")
	errGenesisNotMatch = errors.New("genesis hash mismatch")
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

	curSyncMagic  uint32
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

	if len(headers) > 0 && headers[len(headers)-1].Hash() == chain.CurrentHeader().Hash() {
		sendMsg.HasFinished = true
	}

	buff := common.SerializePanic(sendMsg)
	p.log.Debug("peer send [downloadHeadersResponseCode] query with size %d byte", len(buff))
	return p2p.SendMessage(p.rw, downloadHeadersResponseCode, buff)
}

func (p *peer) sendSyncHashRequest(magic uint32, begin uint64) error {
	sendMsg := &HeaderHashSyncQuery{
		Magic:    magic,
		BeginNum: begin,
	}
	buff := common.SerializePanic(sendMsg)

	p.log.Debug("peer send [syncHashRequestCode] with length: size:%d byte peerid:%s begin=%d", len(buff), p.peerStrID, begin)
	return p2p.SendMessage(p.rw, syncHashRequestCode, buff)
}

// handleSyncHashRequest reponses syncHashRequestCode request, this should only be called by server mode.
func (p *peer) handleSyncHashRequest(msg *HeaderHashSyncQuery) error {
	chain := p.protocolManager.chain
	header := chain.CurrentHeader()
	localTD, err := chain.GetStore().GetBlockTotalDifficulty(header.Hash())
	if err != nil {
		return errReadChain
	}

	height := header.Height
	syncMsg := &HeaderHashSync{
		Magic:           msg.Magic,
		TD:              localTD,
		CurrentBlock:    header.Hash(),
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

	p.log.Debug("peer send [syncHashResponseCode] with length: size:%d byte peerid:%s BeginNum=%d count=%d", len(buff), p.peerStrID, msg.BeginNum, len(headerArr))
	return p2p.SendMessage(p.rw, syncHashResponseCode, buff)
}

// findIdxByHash finds index of hash in p.blockHashArr, and returns -1 if not found
func (p *peer) findIdxByHash(hash common.Hash) int {
	for idx := 0; idx < len(p.blockHashArr); idx++ {
		if p.blockHashArr[idx] == hash {
			return idx
		}
	}

	return -1
}

// getHashByHeight returns header hash of height, only useful in client mode
func (p *peer) getHashByHeight(height uint64) (common.Hash, bool) {
	if height >= p.blockNumBegin && height < (p.blockNumBegin+uint64(len(p.blockHashArr))) {
		return p.blockHashArr[height-p.blockNumBegin], true
	}

	return common.EmptyHash, false
}

// handleSyncHash handles HeaderHashSync message, this should only be called by client mode
func (p *peer) handleSyncHash(msg *HeaderHashSync) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.td, p.head, p.headBlockNum = msg.TD, msg.CurrentBlock, msg.CurrentBlockNum
	if len(msg.HeaderArr) <= 1 {
		// sync finished this round
		p.curSyncMagic = 0
		return nil
	}

	if len(p.blockHashArr) == 0 {
		p.blockNumBegin, p.blockHashArr = msg.BeginNum, msg.HeaderArr
	} else {
		idx := p.findIdxByHash(msg.HeaderArr[0])
		if idx < 0 {
			p.log.Info("handleSyncHash hash not match. %s", p.blockHashArr[0].Hex())
			p.curSyncMagic = 0
			return nil
		}

		p.blockHashArr = append(p.blockHashArr[0:idx], msg.HeaderArr...)
		p.log.Debug("peer handleSyncHash. headBlockNum=%d p.blockNumBegin=%d idx=%d idxheight=%d len(p.blockHashArr)=%d",
			p.headBlockNum, p.blockNumBegin, idx, msg.BeginNum, len(p.blockHashArr))
	}

	lastBlockNum := p.blockNumBegin + uint64(len(p.blockHashArr)) - 1
	if lastBlockNum == p.headBlockNum {
		// sync finished this round
		p.curSyncMagic = 0
		return nil
	}

	p.log.Debug("peer handleSyncHash. need request more. magic=%d startblock=%d len(p.blockHashArr)=%d", p.curSyncMagic, lastBlockNum, len(p.blockHashArr))
	return p.sendSyncHashRequest(p.curSyncMagic, lastBlockNum)
}

func (p *peer) sendAnnounceQuery(magic uint32, begin uint64, end uint64) error {
	query := &AnnounceQuery{
		Magic: magic,
		Begin: begin,
		End:   end,
	}

	buff := common.SerializePanic(query)
	p.log.Debug("peer send [announceRequestCode] query with size %d byte", len(buff))
	return p2p.SendMessage(p.rw, announceRequestCode, buff)
}

// sendAnnounce sends header hash between [begin,end] selectively,
// if end equals 0, end should be maximum block number in blockchain.
func (p *peer) sendAnnounce(magic uint32, begin uint64, end uint64) error {
	chain := p.protocolManager.chain
	if end == 0 {
		end = chain.CurrentHeader().Height
	}

	header := chain.CurrentHeader()
	localTD, err := chain.GetStore().GetBlockTotalDifficulty(header.Hash())
	if err != nil {
		return errReadChain
	}

	height := header.Height
	msg := &AnnounceBody{
		Magic:           magic,
		TD:              localTD,
		CurrentBlock:    header.Hash(),
		CurrentBlockNum: height,
	}

	var numArr []uint64
	var hashArr []common.Hash
	for power2 := uint64(1); ; power2 = power2 * 2 {
		idx, curNum := power2-1, begin
		if end > idx {
			curNum = end - idx

			// must be between begin and end, when curNum less than begin, set it as begin
			if curNum < begin {
				curNum = begin
			}
		}

		curBlock, err := chain.GetStore().GetBlockByHeight(curNum)
		if err != nil {
			p.log.Error("Load block error: %s", err)
			return err
		}

		numArr = append(numArr, curNum)
		hashArr = append(hashArr, curBlock.HeaderHash)

		// if curNum equal begin or cache full break
		if curNum == begin || len(numArr) >= int(MaxGapForAnnounce) {
			break
		}
	}

	msg.BlockNumArr, msg.HeaderArr = numArr, hashArr
	buff := common.SerializePanic(msg)
	p.log.Debug("peer send [announceCode] with magic:%d length:%d bytes peerid:%s num:%d", magic, len(buff), p.peerStrID, len(msg.HeaderArr))

	return p2p.SendMessage(p.rw, announceCode, buff)
}

func (p *peer) handleAnnounce(msg *AnnounceBody) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.curSyncMagic != 0 && p.curSyncMagic != msg.Magic {
		return nil
	}

	if len(msg.HeaderArr) == 0 {
		panic("can not come here")
	}

	p.curSyncMagic = msg.Magic
	p.td, p.head, p.headBlockNum = msg.TD, msg.CurrentBlock, msg.CurrentBlockNum

	startNum, bMatch := uint64(0), false
	if len(p.blockHashArr) == 0 {
		if len(msg.HeaderArr) == 1 {
			// server only has genesis block
			p.blockHashArr = append(p.blockHashArr, msg.HeaderArr[0])
			p.blockNumBegin = msg.BlockNumArr[0]
			p.curSyncMagic = 0
			return nil
		}

		chain := p.protocolManager.chain
		for idx := 0; idx < len(msg.HeaderArr); idx++ {
			height := msg.BlockNumArr[idx]
			hash, err := chain.GetStore().GetBlockHash(height)
			if err != nil {
				continue
			}

			if hash == msg.HeaderArr[idx] {
				startNum, bMatch = height, true
				if idx != (len(msg.HeaderArr)-1) && msg.BlockNumArr[idx+1] > height && (msg.BlockNumArr[idx+1]-height) > MaxGapForAnnounce {
					// send announceRequest message because gap is big enough
					return p.sendAnnounceQuery(p.curSyncMagic, height, msg.BlockNumArr[idx+1])
				}
				break
			}
		}

		if p.headBlockNum-startNum < MinHashesCached {
			if p.headBlockNum > MinHashesCached {
				startNum = p.headBlockNum - MinHashesCached
			} else {
				startNum = uint64(0)
			}
		}
	} else {
		// find common ancestor with peer.blockHashArr
		if len(msg.HeaderArr) == 1 {
			// server only has genesis block
			p.curSyncMagic = 0
			return nil
		}

		for idx := 0; idx < len(msg.HeaderArr); idx++ {
			height := msg.BlockNumArr[idx]
			if cacheHash, bFind := p.getHashByHeight(height); bFind {
				if cacheHash == msg.HeaderArr[idx] {
					startNum, bMatch = height, true
					break
				}
			}
		}

		// todo if not match, should clear local hash, and synchronize again?
		if startNum == p.blockNumBegin+uint64(len(p.blockHashArr))-1 {
			// need not sync
			p.curSyncMagic = 0
			return nil
		}

	}

	if !bMatch {
		panic("can not come here")
	}

	return p.sendSyncHashRequest(p.curSyncMagic, startNum)
}

// handShake exchange networkid td etc between two connected peers.
func (p *peer) handShake(networkID string, td *big.Int, head common.Hash, headBlockNum uint64, genesis common.Hash) error {
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

	if retStatusMsg.NetworkID != networkID {
		return errNetworkNotMatch
	}

	if retStatusMsg.GenesisBlock != genesis {
		return errGenesisNotMatch
	}

	if retStatusMsg.IsServer == p.protocolManager.bServerMode {
		return errModeNotMatch
	}

	p.head, p.td, p.headBlockNum = retStatusMsg.CurrentBlock, retStatusMsg.TD, retStatusMsg.CurrentBlockNum
	return nil
}
