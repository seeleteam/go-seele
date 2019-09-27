/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/hashicorp/golang-lru"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/seele/download"
)

const (
	// DiscHandShakeErr peer handshake error
	DiscHandShakeErr = "disconnect because got handshake error"

	maxKnownTxs    = 25000 // Maximum transactions hashes to keep in the known list
	maxKnownBlocks = 250   // Maximum block hashes to keep in the known list

	// the known debts is not the bigger the better. we should forgot old debt for debt resend.
	// Maximum debt hashes to keep in the known list
	maxKnownDebts = 10000
)

var (
	errMsgNotMatch              = errors.New("Message not match")
	errNetworkNotMatch          = errors.New("NetworkID not match")
	errGenesisNotMatch          = errors.New("Genesis not match")
	errGenesisDifficultNotMatch = errors.New("Genesis Difficult not match")
)

// PeerInfo represents a short summary of a connected peer.
type PeerInfo struct {
	Version    uint     `json:"version"`    // Seele protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // SHA3 hash of the peer's best owned block
}

type peer struct {
	*p2p.Peer
	peerID    common.Address // id of the peer
	peerStrID string
	version   uint // Seele protocol version negotiated
	head      common.Hash
	td        *big.Int // total difficulty
	lock      sync.RWMutex

	rw p2p.MsgReadWriter // the read write method for this peer

	knownTxs    *lru.Cache // Set of transaction hashes known by this peer
	knownBlocks *lru.Cache // Set of block hashes known by this peer
	knownDebts  *lru.Cache // Set of debt hashes known by this peer

	log *log.SeeleLog
}

func idToStr(id common.Address) string {
	return fmt.Sprintf("%x", id[:8])
}

func newPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter, log *log.SeeleLog) *peer {
	knownTxsCache, err := lru.New(maxKnownTxs)
	if err != nil {
		panic(err)
	}

	knownBlockCache, err := lru.New(maxKnownBlocks)
	if err != nil {
		panic(err)
	}

	knownDebtCache, err := lru.New(maxKnownDebts)
	if err != nil {
		panic(err)
	}

	return &peer{
		Peer:        p,
		version:     version,
		td:          big.NewInt(0),
		peerID:      p.Node.ID,
		peerStrID:   idToStr(p.Node.ID),
		knownTxs:    knownTxsCache,
		knownBlocks: knownBlockCache,
		knownDebts:  knownDebtCache,
		rw:          rw,
		log:         log,
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

// Send writes an RLP-encoded message with the given code.
func (p *peer) Send(msgcode uint16, data interface{}) error {
	buff := common.SerializePanic(data)
	return p2p.SendMessage(p.rw, msgcode, buff)
}

func (p *peer) sendTransactionHash(txHash common.Hash) error {
	if p.knownTxs.Contains(txHash) {
		return nil
	}
	buff := common.SerializePanic(txHash)

	err := p2p.SendMessage(p.rw, transactionHashMsgCode, buff)
	if err == nil {
		p.knownTxs.Add(txHash, nil)
	}

	return err
}

func (p *peer) sendDebts(debts []*types.Debt, filter bool) error {
	var filterDebts []*types.Debt
	if filter {
		for _, d := range debts {
			if d != nil && !p.knownDebts.Contains(d.Hash) {
				filterDebts = append(filterDebts, d)
			}
		}
	} else {
		filterDebts = debts
	}

	if len(filterDebts) > 0 {
		buff := common.SerializePanic(filterDebts)
		p.log.Debug("peer send [debtMsgCode] with size %d bytes and %d debts, first debt hash: %v", len(buff), len(filterDebts), filterDebts[0].Hash.Hex())
		err := p2p.SendMessage(p.rw, debtMsgCode, buff)
		if err == nil {
			for _, d := range filterDebts {
				p.knownDebts.Add(d.Hash, nil)
			}
		}

		return err
	}

	return nil
}

func (p *peer) sendTransactionRequest(txHash common.Hash) error {
	buff := common.SerializePanic(txHash)

	return p2p.SendMessage(p.rw, transactionRequestMsgCode, buff)
}

func (p *peer) sendTransaction(tx *types.Transaction) error {
	return p.sendTransactions([]*types.Transaction{tx})
}

func (p *peer) SendBlockHash(blockHash common.Hash) error {
	if p.knownBlocks.Contains(blockHash) {
		return nil
	}
	buff := common.SerializePanic(blockHash)

	p.log.Debug("peer send [blockHashMsgCode] with size %d byte", len(buff))
	err := p2p.SendMessage(p.rw, blockHashMsgCode, buff)
	if err == nil {
		p.knownBlocks.Add(blockHash, nil)
	}

	return err
}

func (p *peer) SendBlockRequest(blockHash common.Hash) error {
	buff := common.SerializePanic(blockHash)

	p.log.Debug("peer send [blockRequestMsgCode] with size %d byte", len(buff))
	return p2p.SendMessage(p.rw, blockRequestMsgCode, buff)
}

func (p *peer) sendTransactions(txs []*types.Transaction) error {
	buff := common.SerializePanic(txs)

	return p2p.SendMessage(p.rw, transactionsMsgCode, buff)
}

func (p *peer) SendBlock(block *types.Block) error {
	buff := common.SerializePanic(block)

	p.log.Debug("peer send [blockMsgCode] with height %d, size %d byte", block.Header.Height, len(buff))
	return p2p.SendMessage(p.rw, blockMsgCode, buff)
}

// Head retrieves a copy of the current head hash and total difficulty.
func (p *peer) Head() (hash common.Hash, td *big.Int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash, new(big.Int).Set(p.td)
}

// SetHead updates the head hash and total difficulty of the peer.
func (p *peer) SetHead(hash common.Hash, td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
	p.td.Set(td)
}

// RequestHeadersByHashOrNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestHeadersByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int, reverse bool) error {
	query := &blockHeadersQuery{
		Magic:   magic,
		Hash:    origin,
		Number:  num,
		Amount:  uint64(amount),
		Reverse: reverse,
	}

	buff := common.SerializePanic(query)
	p.log.Debug("peer send [downloader.GetBlockHeadersMsg] with size %d byte peerid:%s", len(buff), p.peerStrID)
	return p2p.SendMessage(p.rw, downloader.GetBlockHeadersMsg, buff)
}

func (p *peer) sendBlockHeaders(magic uint32, headers []*types.BlockHeader) error {
	sendMsg := &downloader.BlockHeadersMsgBody{
		Magic:   magic,
		Headers: headers,
	}
	buff := common.SerializePanic(sendMsg)

	p.log.Debug("peer send [downloader.BlockHeadersMsg] with length %d size %d byte peerid:%s", len(headers), len(buff), p.peerStrID)
	err := p2p.SendMessage(p.rw, downloader.BlockHeadersMsg, buff)
	if err != nil {
		p.log.Error("peer send [downloader.BlockHeadersMsg] err=%s", err)
	}

	return err
}

// RequestBlocksByHashOrNumber fetches a batch of blocks corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestBlocksByHashOrNumber(magic uint32, origin common.Hash, num uint64, amount int) error {
	query := &blocksQuery{
		Magic:  magic,
		Hash:   origin,
		Number: num,
		Amount: uint64(amount),
	}
	buff := common.SerializePanic(query)

	p.log.Debug("peer send [downloader.GetBlocksMsg] query with size %d byte,peer:%s", len(buff),p.peerStrID)
	return p2p.SendMessage(p.rw, downloader.GetBlocksMsg, buff)
}

func (p *peer) GetPeerRequestInfo() (uint32, common.Hash, uint64, int) {
	return 0, common.EmptyHash, 0, 0
}

func (p *peer) sendBlocks(magic uint32, blocks []*types.Block) error {
	sendMsg := &downloader.BlocksMsgBody{
		Magic:  magic,
		Blocks: blocks,
	}
	buff := common.SerializePanic(sendMsg)

	p.log.Debug("peer send [downloader.BlocksMsg] with length: %d, size:%d byte peerid:%s", len(blocks), len(buff), p.peerStrID)
	err := p2p.SendMessage(p.rw, downloader.BlocksMsg, buff)
	if err != nil {
		p.log.Error("peer send [downloader.BlocksMsg] err=%s", err)
	}

	return err
}

func (p *peer) sendHeadStatus(msg *chainHeadStatus) error {
	buff := common.SerializePanic(msg)

	p.log.Debug("peer send [statusChainHeadMsgCode] with size %d byte", len(buff))
	return p2p.SendMessage(p.rw, statusChainHeadMsgCode, buff)
}

// handShake exchange networkid td etc between two connected peers.
func (p *peer) handShake(networkID string, td *big.Int, head common.Hash, genesis common.Hash, difficult uint64) error {
	msg := &statusData{
		ProtocolVersion: uint32(common.SeeleVersion),
		NetworkID:       networkID,
		TD:              td,
		CurrentBlock:    head,
		GenesisBlock:    genesis,
		Shard:           common.LocalShardNumber,
		Difficult:       difficult,
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

	if err = verifyGenesisAndNetworkID(retStatusMsg, genesis, networkID, common.LocalShardNumber, difficult); err != nil {
		return err
	}

	p.head = retStatusMsg.CurrentBlock
	p.td = retStatusMsg.TD
	return nil
}

func verifyGenesisAndNetworkID(retStatusMsg statusData, genesis common.Hash, networkID string, shard uint, difficult uint64) error {
	if retStatusMsg.NetworkID != networkID {
		return errNetworkNotMatch
	}
	if retStatusMsg.Shard == shard {
		if retStatusMsg.GenesisBlock != genesis {
			return errGenesisNotMatch
		}
	} else {
		if retStatusMsg.Difficult != difficult {
			return errGenesisDifficultNotMatch
		}
	}
	return nil
}

func (p *peer) DisconnectPeer(reason string) {
	p.Peer.Disconnect(reason)
}