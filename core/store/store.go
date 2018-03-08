/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

// BlockchainStore is the interface that wraps the atomic CRUD methods of blockchain.
type BlockchainStore interface {
	// GetBlockHash retrieves the block hash for the specified block height.
	GetBlockHash(height uint64) (common.Hash, error)

	// GetBlockHeader retrieves the block header for the specified block hash.
	GetBlockHeader(hash common.Hash) (*types.BlockHeader, error)
}
