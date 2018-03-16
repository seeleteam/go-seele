/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"encoding/binary"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
)

var (
	keyHeadBlockHash = []byte("HeadBlockHash")

	keyPrefixHash   = []byte("H")
	keyPrefixHeader = []byte("h")
	keyPrefixTD     = []byte("t")
)

type blockchainDatabase struct {
	db database.Database
}

// NewBlockchainDatabase returns a blockchainDatabase instance.
// There are following mappings in database:
//   1) keyPrefixHash + height => hash
//   2) keyHeadBlockHash => HEAD hash
//   3) keyPrefixHeader + hash => header
//   4) keyPrefixTD + hash => total difficulty (td for short)
func NewBlockchainDatabase(db database.Database) BlockchainStore {
	return &blockchainDatabase{db}
}

func heightToHashKey(height uint64) []byte { return append(keyPrefixHash, encodeBlockHeight(height)...) }
func hashToHeaderKey(hash []byte) []byte   { return append(keyPrefixHeader, hash...) }
func hashToTDKey(hash []byte) []byte       { return append(keyPrefixTD, hash...) }

func (store *blockchainDatabase) GetBlockHash(height uint64) (common.Hash, error) {
	hashBytes, err := store.db.Get(heightToHashKey(height))
	if err != nil {
		return common.EmptyHash, err
	}

	return common.BytesToHash(hashBytes), nil
}

// encodeBlockHeight encodes a block height as big endian uint64
func encodeBlockHeight(height uint64) []byte {
	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, height)
	return encoded
}

func (store *blockchainDatabase) GetHeadBlockHash() (common.Hash, error) {
	hashBytes, err := store.db.Get(keyHeadBlockHash)
	if err != nil {
		return common.EmptyHash, err
	}

	return common.BytesToHash(hashBytes), nil
}

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

func (store *blockchainDatabase) PutBlockHeader(hash common.Hash, header *types.BlockHeader, td *big.Int, isHead bool) error {
	if header == nil {
		panic("header is nil.")
	}

	headerBytes, err := common.Serialize(header)
	if err != nil {
		return err
	}

	hashBytes := hash.Bytes()

	batch := store.db.NewBatch()
	batch.Put(heightToHashKey(header.Height), hashBytes)
	batch.Put(hashToHeaderKey(hashBytes), headerBytes)
	batch.Put(hashToTDKey(hashBytes), common.SerializePanic(td))

	if isHead {
		batch.Put(keyHeadBlockHash, hashBytes)
	}

	return batch.Commit()
}

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
