/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
)

var (
	keyHeadBlockHash = []byte("HeadBlockHash")

	keyPrefixHash     = []byte("H")
	keyPrefixHeader   = []byte("h")
	keyPrefixTD       = []byte("t")
	keyPrefixBody     = []byte("b")
	keyPrefixReceipts = []byte("r")
	keyPrefixTxIndex  = []byte("i")
)

// blockBody represents the payload of a block
type blockBody struct {
	Txs []*types.Transaction // Txs is a transaction collection
}

// blockchainDatabase wraps a database used for the blockchain
type blockchainDatabase struct {
	db database.Database
}

// NewBlockchainDatabase returns a blockchainDatabase instance.
// There are following mappings in database:
//   1) keyPrefixHash + height => hash
//   2) keyHeadBlockHash => HEAD hash
//   3) keyPrefixHeader + hash => header
//   4) keyPrefixTD + hash => total difficulty (td for short)
//   5) keyPrefixBody + hash => block body (transactions)
//   6) keyPrefixReceipts + hash => block receipts
//   7) keyPrefixTxIndex + txHash => txIndex
func NewBlockchainDatabase(db database.Database) BlockchainStore {
	return &blockchainDatabase{db}
}

func heightToHashKey(height uint64) []byte  { return append(keyPrefixHash, encodeBlockHeight(height)...) }
func hashToHeaderKey(hash []byte) []byte    { return append(keyPrefixHeader, hash...) }
func hashToTDKey(hash []byte) []byte        { return append(keyPrefixTD, hash...) }
func hashToBodyKey(hash []byte) []byte      { return append(keyPrefixBody, hash...) }
func hashToReceiptsKey(hash []byte) []byte  { return append(keyPrefixReceipts, hash...) }
func txHashToIndexKey(txHash []byte) []byte { return append(keyPrefixTxIndex, txHash...) }

// GetBlockHash gets the hash of the block with the specified height in the blockchain database
func (store *blockchainDatabase) GetBlockHash(height uint64) (common.Hash, error) {
	hashBytes, err := store.db.Get(heightToHashKey(height))
	if err != nil {
		return common.EmptyHash, err
	}

	return common.BytesToHash(hashBytes), nil
}

// PutBlockHash puts the given block height which is encoded as the key
// and hash as the value to the blockchain database.
func (store *blockchainDatabase) PutBlockHash(height uint64, hash common.Hash) error {
	return store.db.Put(heightToHashKey(height), hash.Bytes())
}

// DeleteBlockHash deletes the block hash mapped to by the specified height from the blockchain database
func (store *blockchainDatabase) DeleteBlockHash(height uint64) (bool, error) {
	key := heightToHashKey(height)

	found, err := store.db.Has(key)
	if err != nil {
		return false, err
	}

	if !found {
		return false, nil
	}

	if err = store.db.Delete(key); err != nil {
		return false, err
	}

	return true, nil
}

// encodeBlockHeight encodes a block height as big endian uint64
func encodeBlockHeight(height uint64) []byte {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, height)
	return encoded
}

// GetHeadBlockHash gets the HEAD block hash in the blockchain database
func (store *blockchainDatabase) GetHeadBlockHash() (common.Hash, error) {
	hashBytes, err := store.db.Get(keyHeadBlockHash)
	if err != nil {
		return common.EmptyHash, err
	}

	return common.BytesToHash(hashBytes), nil
}

// PutHeadBlockHash writes the HEAD block hash into the store.
func (store *blockchainDatabase) PutHeadBlockHash(hash common.Hash) error {
	return store.db.Put(keyHeadBlockHash, hash.Bytes())
}

// GetBlockHeader gets the header of the block with the specified hash in the blockchain database
func (store *blockchainDatabase) GetBlockHeader(hash common.Hash) (*types.BlockHeader, error) {
	headerBytes, err := store.db.Get(hashToHeaderKey(hash.Bytes()))
	if err != nil {
		return nil, err
	}

	header := new(types.BlockHeader)
	if err := common.Deserialize(headerBytes, header); err != nil {
		return nil, err
	}

	return header, nil
}

// HasBlock indicates if the block with the specified hash exists in the blockchain database
func (store *blockchainDatabase) HasBlock(hash common.Hash) (bool, error) {
	key := hashToHeaderKey(hash.Bytes())

	found, err := store.db.Has(key)
	if err != nil {
		return false, err
	}

	return found, nil
}

// PutBlockHeader serializes the given block header of the block with the specified hash
// and total difficulty into the blockchain database.
// isHead indicates if the given header is the HEAD block header
func (store *blockchainDatabase) PutBlockHeader(hash common.Hash, header *types.BlockHeader, td *big.Int, isHead bool) error {
	return store.putBlockInternal(hash, header, nil, td, isHead)
}

func (store *blockchainDatabase) putBlockInternal(hash common.Hash, header *types.BlockHeader, body *blockBody, td *big.Int, isHead bool) error {
	if header == nil {
		panic("header is nil")
	}

	headerBytes := common.SerializePanic(header)

	hashBytes := hash.Bytes()

	batch := store.db.NewBatch()
	batch.Put(hashToHeaderKey(hashBytes), headerBytes)
	batch.Put(hashToTDKey(hashBytes), common.SerializePanic(td))

	if body != nil {
		batch.Put(hashToBodyKey(hashBytes), common.SerializePanic(body))

		// Write index for each tx.
		for i, tx := range body.Txs {
			idx := types.TxIndex{BlockHash: hash, Index: uint(i)}
			encodedTxIndex := common.SerializePanic(idx)
			batch.Put(txHashToIndexKey(tx.Hash.Bytes()), encodedTxIndex)
		}
	}

	if isHead {
		batch.Put(heightToHashKey(header.Height), hashBytes)
		batch.Put(keyHeadBlockHash, hashBytes)
	}

	return batch.Commit()
}

// GetBlockTotalDifficulty gets the total difficulty of the block with the specified hash in the blockchain database
func (store *blockchainDatabase) GetBlockTotalDifficulty(hash common.Hash) (*big.Int, error) {
	tdBytes, err := store.db.Get(hashToTDKey(hash.Bytes()))
	if err != nil {
		return nil, err
	}

	td := new(big.Int)
	if err = common.Deserialize(tdBytes, td); err != nil {
		return nil, err
	}

	return td, nil
}

// PutBlock serializes the given block with the specified total difficulty into the blockchain database.
// isHead indicates if the block is the header block
func (store *blockchainDatabase) PutBlock(block *types.Block, td *big.Int, isHead bool) error {
	if block == nil {
		panic("block is nil")
	}

	return store.putBlockInternal(block.HeaderHash, block.Header, &blockBody{block.Transactions}, td, isHead)
}

// GetBlock gets the block with the specified hash in the blockchain database
func (store *blockchainDatabase) GetBlock(hash common.Hash) (*types.Block, error) {
	header, err := store.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}

	bodyKey := hashToBodyKey(hash.Bytes())
	hasBody, err := store.db.Has(bodyKey)
	if err != nil {
		return nil, err
	}

	if !hasBody {
		return &types.Block{
			HeaderHash: hash,
			Header:     header,
		}, nil
	}

	bodyBytes, err := store.db.Get(bodyKey)
	if err != nil {
		return nil, err
	}

	body := blockBody{}
	if err := common.Deserialize(bodyBytes, &body); err != nil {
		return nil, err
	}

	return &types.Block{
		HeaderHash:   hash,
		Header:       header,
		Transactions: body.Txs,
	}, nil
}

// DeleteBlock deletes the block of the specified block hash.
func (store *blockchainDatabase) DeleteBlock(hash common.Hash) error {
	hashBytes := hash.Bytes()
	batch := store.db.NewBatch()

	// delete header, TD and receipts if any.
	headerKey := hashToHeaderKey(hashBytes)
	tdKey := hashToTDKey(hashBytes)
	receiptsKey := hashToReceiptsKey(hashBytes)
	if err := store.delete(batch, headerKey, tdKey, receiptsKey); err != nil {
		return err
	}

	// get body for more deletion
	bodyKey := hashToBodyKey(hashBytes)
	found, err := store.db.Has(bodyKey)
	if err != nil {
		return err
	}

	if !found {
		return batch.Commit()
	}

	encodedBody, err := store.db.Get(bodyKey)
	if err != nil {
		return err
	}

	var body blockBody
	if err = common.Deserialize(encodedBody, &body); err != nil {
		return err
	}

	// delete all tx index in block
	for _, tx := range body.Txs {
		if err = store.delete(batch, txHashToIndexKey(tx.Hash.Bytes())); err != nil {
			return err
		}
	}

	// delete body
	batch.Delete(bodyKey)

	return batch.Commit()
}

func (store *blockchainDatabase) delete(batch database.Batch, keys ...[]byte) error {
	for _, k := range keys {
		found, err := store.db.Has(k)
		if err != nil {
			return err
		}

		if found {
			batch.Delete(k)
		}
	}

	return nil
}

// GetBlockByHeight gets the block with the specified height in the blockchain database
func (store *blockchainDatabase) GetBlockByHeight(height uint64) (*types.Block, error) {
	hash, err := store.GetBlockHash(height)
	if err != nil {
		return nil, err
	}
	block, err := store.GetBlock(hash)
	if err != nil {
		return nil, err
	}
	return block, nil
}

// PutReceipts serializes given receipts for the specified block hash.
func (store *blockchainDatabase) PutReceipts(hash common.Hash, receipts []*types.Receipt) error {
	encodedBytes, err := common.Serialize(receipts)
	if err != nil {
		return err
	}

	key := hashToReceiptsKey(hash.Bytes())

	return store.db.Put(key, encodedBytes)
}

// GetReceiptsByBlockHash retrieves the receipts for the specified block hash.
func (store *blockchainDatabase) GetReceiptsByBlockHash(hash common.Hash) ([]*types.Receipt, error) {
	key := hashToReceiptsKey(hash.Bytes())
	encodedBytes, err := store.db.Get(key)
	if err != nil {
		return nil, err
	}

	receipts := make([]*types.Receipt, 0)
	if err := common.Deserialize(encodedBytes, &receipts); err != nil {
		return nil, err
	}

	return receipts, nil
}

// GetReceiptByTxHash retrieves the receipt for the specified tx hash.
func (store *blockchainDatabase) GetReceiptByTxHash(txHash common.Hash) (*types.Receipt, error) {
	txIndex, err := store.GetTxIndex(txHash)
	if err != nil {
		return nil, err
	}

	receipts, err := store.GetReceiptsByBlockHash(txIndex.BlockHash)
	if err != nil {
		return nil, err
	}

	if uint(len(receipts)) <= txIndex.Index {
		return nil, fmt.Errorf("invalid tx index, txIndex = %v, receiptsLen = %v", *txIndex, len(receipts))
	}

	return receipts[txIndex.Index], nil
}

// GetTxIndex retrieves the tx index for the specified tx hash.
func (store *blockchainDatabase) GetTxIndex(txHash common.Hash) (*types.TxIndex, error) {
	data, err := store.db.Get(txHashToIndexKey(txHash.Bytes()))
	if err != nil {
		return nil, err
	}

	index := &types.TxIndex{}
	if err := common.Deserialize(data, index); err != nil {
		return nil, err
	}

	return index, nil
}
