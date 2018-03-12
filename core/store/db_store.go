/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"encoding/binary"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
)

var (
	keyHeadBlockHash = []byte("HeadBlockHash")
)

type blockchainDatabase struct {
	db database.Database
}

// NewBlockchainDatabase returns a blockchainDatabase instance.
// There are following mappings in database:
//   1) height => hash
//   2) HeadBlockHash => hash
//   3) hash => header
func NewBlockchainDatabase(db database.Database) BlockchainStore {
	return &blockchainDatabase{db}
}

func (store *blockchainDatabase) GetBlockHash(height uint64) (common.Hash, error) {
	hashBytes, err := store.db.Get(encodeBlockHeight(height))
	if err != nil {
		return common.Hash{}, err
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
		return common.Hash{}, err
	}

	return common.BytesToHash(hashBytes), nil
}

func (store *blockchainDatabase) GetBlockHeader(hash common.Hash) (*types.BlockHeader, error) {
	headerBytes, err := store.db.Get(hash.Bytes())
	if err != nil {
		return nil, err
	}

	header := new(types.BlockHeader)
	if err := common.Deserialize(headerBytes, header); err != nil {
		return nil, err
	}

	return header, nil
}

func (store *blockchainDatabase) PutBlockHeader(header *types.BlockHeader, isHead bool) error {
	if header == nil {
		panic("header is nil.")
	}

	headerBytes, err := common.Serialize(header)
	if err != nil {
		return err
	}

	headerHash := crypto.Keccak256Hash(headerBytes)

	batch := store.db.NewBatch()
	batch.Put(encodeBlockHeight(header.Height.Uint64()), headerHash)
	batch.Put(headerHash, headerBytes)

	if isHead {
		batch.Put(keyHeadBlockHash, headerHash)
	}

	return batch.Commit()
}
