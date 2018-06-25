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
	// GetBlockHash retrieves the block hash for the specified canonical block height.
	GetBlockHash(height uint64) (common.Hash, error)

	// PutBlockHash writes the height-to-blockHash entry in the canonical chain.
	PutBlockHash(height uint64, hash common.Hash) error

	// DeleteBlockHash deletes the block hash of the specified canonical block height.
	DeleteBlockHash(height uint64) (bool, error)

	// GetHeadBlockHash retrieves the HEAD block hash.
	GetHeadBlockHash() (common.Hash, error)

	// PutHeadBlockHash writes the HEAD block hash into the store.
	PutHeadBlockHash(hash common.Hash) error

	// GetBlockHeader retrieves the block header for the specified block hash.
	GetBlockHeader(hash common.Hash) (*types.BlockHeader, error)

	// PutBlockHeader serializes a block header with the specified total difficulty (td) into the store.
	// The input parameter isHead indicates if the header is a HEAD block header.
	PutBlockHeader(hash common.Hash, header *types.BlockHeader, td *big.Int, isHead bool) error

	// GetBlockTotalDifficulty retrieves a block's total difficulty for the specified block hash.
	GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error)

	// PutBlock serializes the given block with the given total difficulty (td) into the store.
	// The input parameter isHead indicates if the given block is a HEAD block.
	PutBlock(block *types.Block, td *big.Int, isHead bool) error

	// GetBlock retrieves the block for the specified block hash.
	GetBlock(hash common.Hash) (*types.Block, error)

	// HasBlock checks if the block with the specified hash exists.
	HasBlock(hash common.Hash) (bool, error)

	// DeleteBlock deletes the block of the specified block hash.
	DeleteBlock(hash common.Hash) error

	// GetBlockByHeight retrieves the block for the specified block height.
	GetBlockByHeight(height uint64) (*types.Block, error)

	// PutReceipts serializes given receipts for the specified block hash.
	PutReceipts(hash common.Hash, receipts []*types.Receipt) error

	// GetReceiptsByBlockHash retrieves the receipts for the specified block hash.
	GetReceiptsByBlockHash(hash common.Hash) ([]*types.Receipt, error)

	// GetReceiptByTxHash retrieves the receipt for the specified tx hash.
	GetReceiptByTxHash(txHash common.Hash) (*types.Receipt, error)

	// GetTxIndex retrieves the tx index for the specified tx hash.
	GetTxIndex(txHash common.Hash) (*types.TxIndex, error)
}
