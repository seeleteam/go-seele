/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"bytes"
	"math/big"

	"github.com/syndtr/goleveldb/leveldb/errors"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

// ErrGenesisHashMismatch is returned when genesis block hash mismatch between store and memory.
var ErrGenesisHashMismatch = errors.New("genesis block hash mismatch")

// Genesis represents the genesis block in the blockchain.
type Genesis struct {
	bcStore store.BlockchainStore
	header  *types.BlockHeader
}

// NewGenesis returns a genesis block with specified block header.
func NewGenesis(bcStore store.BlockchainStore, header *types.BlockHeader) *Genesis {
	return &Genesis{bcStore, header.Clone()}
}

// DefaultGenesis returns the default genesis block in the blockchain.
// TODO default genesis value is TBD according to the consensus algorithm.
func DefaultGenesis(bcStore store.BlockchainStore) *Genesis {
	return &Genesis{
		bcStore: bcStore,
		header: &types.BlockHeader{
			PreviousBlockHash: common.Hash{},
			Creator:           common.Address{},
			TxHash:            common.Hash{},
			Difficulty:        big.NewInt(1),
			Height:            big.NewInt(0),
			CreateTimestamp:   big.NewInt(0),
			Nonce:             1,
		},
	}
}

// Initialize writes the genesis block in blockchain store if unavailable.
// Otherwise, check if the existing genesis block is valid in blockchain store.
func (genesis *Genesis) Initialize() error {
	storedGenesisHash, err := genesis.bcStore.GetBlockHash(0)

	// TODO use seele defined common error instead of levelDB error.
	if err == errors.ErrNotFound {
		return genesis.store()
	}

	headerBytes := common.SerializePanic(genesis.header)
	headerHash := crypto.Keccak256Hash(headerBytes)
	if !bytes.Equal(headerHash, storedGenesisHash.Bytes()) {
		return ErrGenesisHashMismatch
	}

	return nil
}

// store atomically stores the genesis block in blockchain store.
func (genesis *Genesis) store() error {
	// TODO note, default accounts should be initialized/stored as well.
	return nil
}
