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

	transactionHashMsgCode uint16 = 0
	blockHashMsgCode       uint16 = 1

	transactionRequestMsgCode uint16 = 2
	transactionMsgCode        uint16 = 3

	blockRequestMsgCode uint16 = 4
	blockMsgCode        uint16 = 5
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

func (p *peer) SendTransactionHash(tx *types.Transaction) error {
	if p.knownTxs.Has(tx.Hash) {
		return nil
	}

	err := p2p.SendMessage(p.rw, transactionHashMsgCode, common.SerializePanic(tx.Hash))
	if err == nil {
		p.knownTxs.Add(tx.Hash)
	}

	return err
}

func (p *peer) SendBlockHash(block *types.Block) error {
	if p.knownBlocks.Has(block.HeaderHash) {
		return nil
	}

	err := p2p.SendMessage(p.rw, blockHashMsgCode, common.SerializePanic(block.HeaderHash))
	if err == nil {
		p.knownBlocks.Add(block.HeaderHash)
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

// HandShake exchange networkid td etc between two connected peers.
func (p *peer) HandShake() error {
	//TODO add exchange status msg
	return nil
}
