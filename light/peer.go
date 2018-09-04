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
)

// PeerInfo represents a short summary of a connected peer.
type PeerInfo struct {
	Version    uint     `json:"version"`    // Seele protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // SHA3 hash of the peer's best owned block
}

type peer struct {
	*p2p.Peer
	peerStrID       string
	peerID          common.Address
	version         uint // Seele protocol version negotiated
	head            common.Hash
	td              *big.Int // total difficulty
	lock            sync.RWMutex
	bSynching       bool // is synchnorising block hash
	protocolManager *LightProtocol
	rw              p2p.MsgReadWriter // the read write method for this peer

	log *log.SeeleLog
}

func idToStr(id common.Address) string {
	return fmt.Sprintf("%x", id[:8])
}

func newPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter, log *log.SeeleLog, protocolManager *LightProtocol) *peer {

	return &peer{
		Peer:            p,
		version:         version,
		td:              big.NewInt(0),
		peerStrID:       idToStr(p.Node.ID),
		peerID:          p.Node.ID,
		rw:              rw,
		protocolManager: protocolManager,
		log:             log,
	}
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

func (p *peer) sendBlock(reqID uint32, block *types.Block) error {
	sendMsg := &BlockMsgBody{
		ReqID: reqID,
		Block: block,
	}
	buff := common.SerializePanic(sendMsg)

	p.log.Debug("peer send [blockMsgCode] with length: size:%d byte peerid:%s", len(buff), p.peerStrID)
	return p2p.SendMessage(p.rw, blockMsgCode, buff)
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

func (p *peer) handleAnnounce(msg *Announce) {
	// todo update local block header hashes
}

// handShake exchange networkid td etc between two connected peers.
func (p *peer) handShake(networkID uint64, td *big.Int, head common.Hash, genesis common.Hash) error {
	msg := &statusData{
		ProtocolVersion: uint32(LightSeeleVersion),
		NetworkID:       networkID,
		IsServer:        p.protocolManager.bServerMode,
		TD:              td,
		CurrentBlock:    head,
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

	p.head = retStatusMsg.CurrentBlock
	p.td = retStatusMsg.TD
	return nil
}
