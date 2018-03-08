/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package store

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
)

type blockchainDatabase struct {
	db database.Database
}

// NewBlockchainDatabase returns a blockchainDatabase instance.
func NewBlockchainDatabase(db database.Database) BlockchainStore {
	return &blockchainDatabase{db}
}

func (db *blockchainDatabase) GetBlockHash(height uint64) (common.Hash, error) {
	// TODO
	return common.Hash{}, nil
}

func (db *blockchainDatabase) GetBlockHeader(hash common.Hash) (*types.BlockHeader, error) {
	// TODO
	return nil, nil
}
