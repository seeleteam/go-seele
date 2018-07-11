/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"math/big"

	"github.com/hashicorp/golang-lru"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

const (
	hashCacheSize   = 1024 // maximum 40K
	headerCacheSize = 512  // maximum 100K
	tdCacheSize     = 1024 // maximum 64K
	blockCacheSize  = 64   // maximum 64M
)

// cachedStore is used to cache recent accessed data to avoid frequent data deserialization.
type cachedStore struct {
	raw BlockchainStore

	hashCache   *lru.Cache // canonical blockchain height to hash cache.
	headerCache *lru.Cache // block hash to header cache.
	tdCache     *lru.Cache // block hash to total difficulty cache.
	blockCache  *lru.Cache // block hash to block cache.
}

// NewCachedStore returns a cached blockchainDatabase instance based on LRU.
func NewCachedStore(store BlockchainStore) BlockchainStore {
	return &cachedStore{
		raw:         store,
		hashCache:   common.MustNewCache(hashCacheSize),
		headerCache: common.MustNewCache(headerCacheSize),
		tdCache:     common.MustNewCache(tdCacheSize),
		blockCache:  common.MustNewCache(blockCacheSize),
	}
}

// GetBlockHash retrieves the block hash for the specified canonical block height.
func (store *cachedStore) GetBlockHash(height uint64) (common.Hash, error) {
	if hash, found := store.hashCache.Get(height); found {
		return hash.(common.Hash), nil
	}

	hash, err := store.raw.GetBlockHash(height)
	if err == nil {
		store.hashCache.Add(height, hash)
	}

	return hash, err
}

// PutBlockHash writes the height-to-blockHash entry in the canonical chain.
func (store *cachedStore) PutBlockHash(height uint64, hash common.Hash) error {
	err := store.raw.PutBlockHash(height, hash)
	if err == nil {
		store.hashCache.Add(height, hash)
	}

	return err
}

// DeleteBlockHash deletes the block hash of the specified canonical block height.
func (store *cachedStore) DeleteBlockHash(height uint64) (bool, error) {
	store.hashCache.Remove(height)
	return store.raw.DeleteBlockHash(height)
}

// GetHeadBlockHash retrieves the HEAD block hash.
func (store *cachedStore) GetHeadBlockHash() (common.Hash, error) {
	return store.raw.GetHeadBlockHash()
}

// PutHeadBlockHash writes the HEAD block hash into the store.
func (store *cachedStore) PutHeadBlockHash(hash common.Hash) error {
	return store.raw.PutHeadBlockHash(hash)
}

// GetBlockHeader retrieves the block header for the specified block hash.
func (store *cachedStore) GetBlockHeader(hash common.Hash) (*types.BlockHeader, error) {
	if header, found := store.headerCache.Get(hash); found {
		return header.(*types.BlockHeader), nil
	}

	header, err := store.raw.GetBlockHeader(hash)
	if err == nil {
		store.headerCache.Add(hash, header)
	}

	return header, err
}

// PutBlockHeader serializes a block header with the specified total difficulty (td) into the store.
// The input parameter isHead indicates if the header is a HEAD block header.
func (store *cachedStore) PutBlockHeader(hash common.Hash, header *types.BlockHeader, td *big.Int, isHead bool) error {
	err := store.raw.PutBlockHeader(hash, header, td, isHead)
	if err == nil {
		store.headerCache.Add(hash, header)
		store.tdCache.Add(hash, td)

		if isHead {
			store.hashCache.Add(header.Height, hash)
		}
	}

	return err
}

// GetBlockTotalDifficulty retrieves a block's total difficulty for the specified block hash.
func (store *cachedStore) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	if td, found := store.tdCache.Get(hash); found {
		return td.(*big.Int), nil
	}

	td, err := store.raw.GetBlockTotalDifficulty(hash)
	if err == nil {
		store.tdCache.Add(hash, td)
	}

	return td, err
}

// PutBlock serializes the given block with the given total difficulty (td) into the store.
// The input parameter isHead indicates if the given block is a HEAD block.
func (store *cachedStore) PutBlock(block *types.Block, td *big.Int, isHead bool) error {
	err := store.raw.PutBlock(block, td, isHead)
	if err == nil {
		store.headerCache.Add(block.HeaderHash, block.Header)
		store.tdCache.Add(block.HeaderHash, td)
		store.blockCache.Add(block.HeaderHash, block)

		if isHead {
			store.hashCache.Add(block.Header.Height, block.HeaderHash)
		}
	}

	return err
}

// GetBlock retrieves the block for the specified block hash.
func (store *cachedStore) GetBlock(hash common.Hash) (*types.Block, error) {
	if block, found := store.blockCache.Get(hash); found {
		return block.(*types.Block), nil
	}

	block, err := store.raw.GetBlock(hash)
	if err == nil {
		store.blockCache.Add(hash, block)
	}

	return block, err
}

// HasBlock checks if the block with the specified hash exists.
func (store *cachedStore) HasBlock(hash common.Hash) (bool, error) {
	if store.headerCache.Contains(hash) {
		return true, nil
	}

	return store.raw.HasBlock(hash)
}

// DeleteBlock deletes the block of the specified block hash.
func (store *cachedStore) DeleteBlock(hash common.Hash) error {
	// remove height-to-hash cache in canonical chain.
	header, err := store.raw.GetBlockHeader(hash)
	if err != nil {
		return err
	}

	canonicalHash, err := store.GetBlockHash(header.Height)
	if err != nil {
		return err
	}

	if canonicalHash.Equal(hash) {
		store.hashCache.Remove(header.Height)
	}

	// remove other caches: header, td and block
	store.headerCache.Remove(hash)
	store.tdCache.Remove(hash)
	store.blockCache.Remove(hash)

	return store.raw.DeleteBlock(hash)
}

// GetBlockByHeight retrieves the block for the specified block height.
func (store *cachedStore) GetBlockByHeight(height uint64) (*types.Block, error) {
	return store.raw.GetBlockByHeight(height)
}

// PutReceipts serializes given receipts for the specified block hash.
func (store *cachedStore) PutReceipts(hash common.Hash, receipts []*types.Receipt) error {
	return store.raw.PutReceipts(hash, receipts)
}

// GetReceiptsByBlockHash retrieves the receipts for the specified block hash.
func (store *cachedStore) GetReceiptsByBlockHash(hash common.Hash) ([]*types.Receipt, error) {
	return store.raw.GetReceiptsByBlockHash(hash)
}

// GetReceiptByTxHash retrieves the receipt for the specified tx hash.
func (store *cachedStore) GetReceiptByTxHash(txHash common.Hash) (*types.Receipt, error) {
	return store.raw.GetReceiptByTxHash(txHash)
}

// GetTxIndex retrieves the tx index for the specified tx hash.
func (store *cachedStore) GetTxIndex(txHash common.Hash) (*types.TxIndex, error) {
	return store.raw.GetTxIndex(txHash)
}
