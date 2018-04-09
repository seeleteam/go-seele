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

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/seele/download"
	set "gopkg.in/fatih/set.v0"
)

const (
	// DiscHandShakeErr peer handshake error
	DiscHandShakeErr = 100

	maxKnownTxs    = 32768 // Maximum transactions hashes to keep in the known list
	maxKnownBlocks = 1024  // Maximum block hashes to keep in the known list
)

var (
	errMsgNotMatch     = errors.New("Message not match")
	errNetworkNotMatch = errors.New("NetworkID not match")
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

	knownTxs    *set.Set // Set of transaction hashes known by this peer
	knownBlocks *set.Set // Set of block hashes known by this peer
}

func newPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return &peer{
		Peer:        p,
		version:     version,
		td:          big.NewInt(0),
		peerID:      p.Node.ID,
		peerStrID:   fmt.Sprintf("%x", p.Node.ID[:8]),
		knownTxs:    set.New(),
		knownBlocks: set.New(),
		rw:          rw,
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

func (p *peer) markTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTxs.Size() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash)
}

func (p *peer) SendTransactionHash(txHash common.Hash) error {
	if p.knownTxs.Has(txHash) {
		return nil
	}

	err := p2p.SendMessage(p.rw, transactionHashMsgCode, common.SerializePanic(txHash))
	if err == nil {
		p.knownTxs.Add(txHash)
	}

	return err
}

func (p *peer) sendTransaction(tx *types.Transaction) error {
	if p.knownTxs.Has(tx.Hash) {
		return nil
	}
	return p2p.SendMessage(p.rw, transactionsMsgCode, common.SerializePanic([]*types.Transaction{tx}))
}

func (p *peer) SendBlockHash(blockHash common.Hash) error {
	if p.knownBlocks.Has(blockHash) {
		return nil
	}

	err := p2p.SendMessage(p.rw, blockHashMsgCode, common.SerializePanic(blockHash))
	if err == nil {
		p.knownBlocks.Add(blockHash)
	}

	return err
}

func (p *peer) SendBlockRequest(blockHash common.Hash) error {
	return p2p.SendMessage(p.rw, blockRequestMsgCode, common.SerializePanic(blockHash))
}

func (p *peer) sendTransactions(txs []*types.Transaction) error {
	return p2p.SendMessage(p.rw, transactionsMsgCode, common.SerializePanic(txs))
}

func (p *peer) SendBlock(block *types.Block) error {
	return p2p.SendMessage(p.rw, blockMsgCode, common.SerializePanic(block))
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
func (p *peer) RequestHeadersByHashOrNumber(origin common.Hash, num uint64, amount int, reverse bool) error {
	query := &blockHeadersQuery{
		Hash:    origin,
		Number:  num,
		Amount:  amount,
		Reverse: reverse,
	}
	return p2p.SendMessage(p.rw, downloader.GetBlockHeadersMsg, common.SerializePanic(query))
}

func (p *peer) sendBlockHeaders(headers []*types.BlockHeader) error {
	return p2p.SendMessage(p.rw, downloader.BlockHeadersMsg, common.SerializePanic(headers))
}

// RequestBlocksByHashOrNumber fetches a batch of blocks corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestBlocksByHashOrNumber(origin common.Hash, num uint64, amount int) error {
	query := &blocksQuery{
		Hash:   origin,
		Number: num,
		Amount: amount,
	}
	return p2p.SendMessage(p.rw, downloader.GetBlocksMsg, common.SerializePanic(query))
}

func (p *peer) sendPreBlocksMsg(numL []uint64) error {
	return p2p.SendMessage(p.rw, downloader.BlocksPreMsg, common.SerializePanic(numL))
}

func (p *peer) sendBlocks(blocks []*types.Block) error {
	return p2p.SendMessage(p.rw, downloader.BlocksMsg, common.SerializePanic(blocks))
}

func (p *peer) sendHeadStatus(msg *chainHeadStatus) error {
	return p2p.SendMessage(p.rw, statusChainHeadMsgCode, common.SerializePanic(msg))
}

// HandShake exchange networkid td etc between two connected peers.
func (p *peer) HandShake(networkID uint64, td *big.Int, head common.Hash, genesis common.Hash) error {
	msg := &statusData{
		ProtocolVersion: uint32(SeeleVersion),
		NetworkID:       networkID,
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

	p.head = retStatusMsg.CurrentBlock
	p.td = retStatusMsg.TD
	return nil
}
