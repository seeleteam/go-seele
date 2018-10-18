/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"errors"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	leveldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
)

var errNotFound = leveldbErrors.ErrNotFound
var ErrDBCorrupt = errors.New("db corrupted")

type memBlock struct {
	td       *big.Int
	block    *types.Block
	receipts []*types.Receipt
}

// MemStore prepresents a in-memory database that used for the blockchain.
type MemStore struct {
	CanonicalBlocks map[uint64]common.Hash // height to block hash map in canonical chain
	HeadBlockHash   common.Hash            // HEAD block hash
	Blocks          map[common.Hash]*memBlock
	TxLookups       map[common.Hash]types.TxIndex   // tx hash to index mapping
	DebtLookups     map[common.Hash]types.DebtIndex // debt hash to index mapping

	CorruptOnPutBlock bool // used to test blockchain recovery if program crashed
}

func NewMemStore() *MemStore {
	return &MemStore{
		CanonicalBlocks: make(map[uint64]common.Hash),
		Blocks:          make(map[common.Hash]*memBlock),
		TxLookups:       make(map[common.Hash]types.TxIndex),
		DebtLookups:     make(map[common.Hash]types.DebtIndex),
	}
}

func (store *MemStore) GetBlockHash(height uint64) (common.Hash, error) {
	hash, found := store.CanonicalBlocks[height]
	if found {
		return hash, nil
	}

	return common.EmptyHash, errNotFound
}

func (store *MemStore) PutBlockHash(height uint64, hash common.Hash) error {
	store.CanonicalBlocks[height] = hash
	return nil
}

func (store *MemStore) DeleteBlockHash(height uint64) (bool, error) {
	if _, found := store.CanonicalBlocks[height]; !found {
		return false, nil
	}

	delete(store.CanonicalBlocks, height)
	return true, nil
}

func (store *MemStore) GetHeadBlockHash() (common.Hash, error) {
	return store.HeadBlockHash, nil
}

func (store *MemStore) PutHeadBlockHash(hash common.Hash) error {
	store.HeadBlockHash = hash
	return nil
}

func (store *MemStore) GetBlockHeader(hash common.Hash) (*types.BlockHeader, error) {
	block := store.Blocks[hash]
	if block == nil {
		return nil, errNotFound
	}

	return block.block.Header, nil
}

func (store *MemStore) PutBlockHeader(hash common.Hash, header *types.BlockHeader, td *big.Int, isHead bool) error {
	block := &types.Block{
		Header:     header,
		HeaderHash: header.Hash(),
	}

	store.Blocks[hash] = &memBlock{
		block: block,
		td:    td,
	}

	if isHead {
		store.CanonicalBlocks[header.Height] = hash
		store.HeadBlockHash = hash
	}

	return nil
}

func (store *MemStore) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	block := store.Blocks[hash]
	if block == nil {
		return nil, errNotFound
	}

	return block.td, nil
}

func (store *MemStore) PutBlock(block *types.Block, td *big.Int, isHead bool) error {
	storedBlock := store.Blocks[block.HeaderHash]
	if storedBlock == nil {
		storedBlock = &memBlock{
			td:    td,
			block: block,
		}
	} else {
		storedBlock.block = block
		storedBlock.td = td
	}

	store.Blocks[block.HeaderHash] = storedBlock

	if store.CorruptOnPutBlock {
		return ErrDBCorrupt
	}

	if isHead {
		store.CanonicalBlocks[block.Header.Height] = block.HeaderHash
		store.HeadBlockHash = block.HeaderHash

		if err := store.AddIndices(block); err != nil {
			return err
		}
	}

	return nil
}

func (store *MemStore) GetBlock(hash common.Hash) (*types.Block, error) {
	if block := store.Blocks[hash]; block != nil {
		return block.block, nil
	}

	return nil, errNotFound
}

func (store *MemStore) HasBlock(hash common.Hash) (bool, error) {
	return store.Blocks[hash] != nil, nil
}

func (store *MemStore) DeleteBlock(hash common.Hash) error {
	if block := store.Blocks[hash]; block != nil {
		for _, tx := range block.block.Transactions {
			delete(store.TxLookups, tx.Hash)
		}

		for _, d := range block.block.Debts {
			delete(store.DebtLookups, d.Hash)
		}
	}

	delete(store.Blocks, hash)

	return nil
}

func (store *MemStore) GetBlockByHeight(height uint64) (*types.Block, error) {
	hash, err := store.GetBlockHash(height)
	if err != nil {
		return nil, err
	}

	return store.GetBlock(hash)
}

func (store *MemStore) PutReceipts(hash common.Hash, receipts []*types.Receipt) error {
	block := store.Blocks[hash]
	if block == nil {
		block = &memBlock{}
		store.Blocks[hash] = block
	}

	block.receipts = receipts

	return nil
}

func (store *MemStore) GetReceiptsByBlockHash(hash common.Hash) ([]*types.Receipt, error) {
	block := store.Blocks[hash]
	if block == nil {
		return nil, errNotFound
	}

	return block.receipts, nil
}

func (store *MemStore) GetReceiptByTxHash(txHash common.Hash) (*types.Receipt, error) {
	txIndex, found := store.TxLookups[txHash]
	if !found {
		return nil, errNotFound
	}

	receipts, err := store.GetReceiptsByBlockHash(txIndex.BlockHash)
	if err != nil {
		return nil, err
	}

	if uint(len(receipts)) <= txIndex.Index {
		return nil, errNotFound
	}

	return receipts[txIndex.Index], nil
}

func (store *MemStore) AddIndices(block *types.Block) error {
	for i, tx := range block.Transactions {
		store.TxLookups[tx.Hash] = types.TxIndex{BlockHash: block.HeaderHash, Index: uint(i)}
	}

	for i, d := range block.Debts {
		store.DebtLookups[d.Hash] = types.DebtIndex{BlockHash: block.HeaderHash, Index: uint(i)}
	}

	return nil
}

func (store *MemStore) GetTxIndex(txHash common.Hash) (*types.TxIndex, error) {
	txIndex, found := store.TxLookups[txHash]
	if !found {
		return nil, errNotFound
	}

	return &txIndex, nil
}

func (store *MemStore) GetDebtIndex(txHash common.Hash) (*types.DebtIndex, error) {
	debtIndex, found := store.DebtLookups[txHash]
	if !found {
		return nil, errNotFound
	}

	return &debtIndex, nil
}

func (store *MemStore) DeleteIndices(block *types.Block) error {
	for _, tx := range block.Transactions {
		if txIdx := store.TxLookups[tx.Hash]; txIdx.BlockHash.Equal(block.HeaderHash) {
			delete(store.TxLookups, tx.Hash)
		}
	}

	for _, debt := range block.Debts {
		if debtIdx := store.DebtLookups[debt.Hash]; debtIdx.BlockHash.Equal(block.HeaderHash) {
			delete(store.DebtLookups, debt.Hash)
		}
	}

	return nil
}
