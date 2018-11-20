/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package consensus

import (
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/common"
)

type Engine interface {
	// Prepare header before generate block
	Prepare(chain ChainReader, header *types.BlockHeader) error

	// VerifyHeader verify block header
	VerifyHeader(chain ChainReader, header *types.BlockHeader) error

	// Seal generate block
	Seal(chain ChainReader, block *types.Block, stop <-chan struct{}, results chan<- *types.Block) error

	// APIs returns the RPC APIs this consensus engine provides.
	APIs(chain ChainReader) []rpc.API

	// SetThreads set miner threads
	SetThreads(thread int)
}

// Istanbul is a consensus engine to avoid byzantine failure
type Istanbul interface {
	Engine

	// Start starts the engine
	Start(chain ChainReader, currentBlock func() *types.Block, hasBadBlock func(hash common.Hash) bool) error

	// Stop stops the engine
	Stop() error
}

// Broadcaster defines the interface to enqueue blocks to fetcher and find peer
type Broadcaster interface {
	// Enqueue add a block into fetcher queue
	Enqueue(id string, block *types.Block)
	// FindPeers retrives peers by addresses
	FindPeers(map[common.Address]bool) map[common.Address]Peer
}

// Peer defines the interface to communicate with peer
type Peer interface {
	// Send sends the message to this peer
	Send(msgcode uint64, data interface{}) error
}

// Protocol defines the protocol of the consensus
type Protocol struct {
	// Official short name of the protocol used during capability negotiation.
	Name string
	// Supported versions of the eth protocol (first is primary).
	Versions []uint
	// Number of implemented message corresponding to different protocol versions.
	Lengths []uint64
}

// ChainReader defines a small collection of methods needed to access the local
// blockchain during header and/or uncle verification.
type ChainReader interface {
	// CurrentHeader retrieves the current header from the local chain.
	CurrentHeader() *types.BlockHeader

	// GetHeaderByNumber retrieves a block header from the database by number.
	GetHeaderByHeight(height uint64) *types.BlockHeader

	// GetHeaderByHash retrieves a block header from the database by its hash.
	GetHeaderByHash(hash common.Hash) *types.BlockHeader

	// GetBlock retrieves a block from the database by hash and number.
	GetBlockByHash(hash common.Hash) *types.Block
}

