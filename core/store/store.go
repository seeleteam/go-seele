/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

// BlockchainStore is the interface that wraps the atomic CRUD methods of blockchain.
type BlockchainStore interface {
	// GetBlockHash retrieves the block hash for the specified block height.
	GetBlockHash(height uint64) (common.Hash, error)
	// GetHeadBlockHash retrieves the HEAD block hash.
	GetHeadBlockHash() (common.Hash, error)

	// GetBlockHeader retrieves the block header for the specified block hash.
	GetBlockHeader(hash common.Hash) (*types.BlockHeader, error)
	// PutBlockHeader serializes a block header with total difficulty into the store.
	// The input parameter isHead indicates if the header is a HEAD block header.
	PutBlockHeader(hash common.Hash, header *types.BlockHeader, td *big.Int, isHead bool) error

	// GetBlockTotalDifficulty retrieves a block's total difficulty for the specified block hash.
	GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error)
}
