/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"encoding/hex"
	"math/big"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/p2p"
	set "gopkg.in/fatih/set.v0"
)

const (
	// DiscHandShakeErr peer handshake error
	DiscHandShakeErr = 100
)

// PeerInfo represents a short summary of a connected peer.
type PeerInfo struct {
	Version    uint     `json:"version"`    // Seele protocol version negotiated
	Difficulty *big.Int `json:"difficulty"` // Total difficulty of the peer's blockchain
	Head       string   `json:"head"`       // SHA3 hash of the peer's best owned block
}

type peer struct {
	*p2p.Peer
	peerID  common.Address // id of the peer
	version uint           // Seele protocol version negotiated
	head    common.Hash
	td      *big.Int // total difficulty
	lock    sync.RWMutex

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

func (p *peer) SendTransactionRequest(txHash common.Hash) error {
	return p2p.SendMessage(p.rw, transactionRequestMsgCode, common.SerializePanic(txHash))
}

func (p *peer) SendBlockRequest(blockHash common.Hash) error {
	return p2p.SendMessage(p.rw, blockRequestMsgCode, common.SerializePanic(blockHash))
}

func (p *peer) SendTransaction(tx *types.Transaction) error {
	return p2p.SendMessage(p.rw, transactionMsgCode, common.SerializePanic(tx))
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
	//TODO send GetBlockHeadersMsg
	return nil
}

// RequestBlocksByHashOrNumber fetches a batch of blocks corresponding to the
// specified header query, based on the hash of an origin block.
func (p *peer) RequestBlocksByHashOrNumber(origin common.Hash, num uint64, amount int) error {
	//TODO send GetBlocksMsg
	return nil
}

func (p *peer) sendNewBlockHash(block *types.Block) {
	// TODO
}

func (p *peer) sendNewBlock(block *types.Block, td *big.Int) {
	// TODO
}

func (p *peer) sendTransactions([]*types.Transaction) error {
	// TODO
	return nil
}

// HandShake exchange networkid td etc between two connected peers.
func (p *peer) HandShake() error {
	//TODO add exchange status msg
	return nil
}
