/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"errors"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

var (
	// ErrHeaderChainInvalidParentHash is returned when insert a new header with invalid parent block hash.
	ErrHeaderChainInvalidParentHash = errors.New("invalid parent block hash")
	// ErrHeaderChainInvalidHeight is returned when insert a new header with invalid block height.
	ErrHeaderChainInvalidHeight = errors.New("invalid block height")
)

// HeaderChain represents the block header chain that is shared by archive node and light node.
// This is a non-thread safe structure.
type HeaderChain struct {
	bcStore store.BlockchainStore

	genesisHeader     *types.BlockHeader
	currentHeader     *types.BlockHeader
	currentHeaderHash common.Hash
}

// NewHeaderChain returns a new instance of HeaderChain structure.
func NewHeaderChain(bcStore store.BlockchainStore) (*HeaderChain, error) {
	hc := HeaderChain{
		bcStore: bcStore,
	}

	// Get genesis block header from store.
	genesisHash, err := bcStore.GetBlockHash(0)
	if err != nil {
		return nil, err
	}

	hc.genesisHeader, err = bcStore.GetBlockHeader(genesisHash)
	if err != nil {
		return nil, err
	}

	// Get HEAD block header from store.
	hc.currentHeaderHash, err = bcStore.GetHeadBlockHash()
	if err != nil {
		return nil, err
	}

	hc.currentHeader, err = bcStore.GetBlockHeader(hc.currentHeaderHash)
	if err != nil {
		return nil, err
	}

	return &hc, nil
}

// WriteHeader writes a block new header into the header chain.
// It requires the new header's parent header is the HEAD header
// in the chain.
func (hc *HeaderChain) WriteHeader(newHeader *types.BlockHeader) error {
	if !newHeader.PreviousBlockHash.Equal(hc.currentHeaderHash) {
		return ErrHeaderChainInvalidParentHash
	}

	if newHeader.Height != hc.currentHeader.Height+1 {
		return ErrHeaderChainInvalidHeight
	}

	// TODO validate the nonce via consensus engine.

	currentTd, err := hc.bcStore.GetBlockTotalDifficulty(hc.currentHeaderHash)
	if err != nil {
		return err
	}

	newTd := new(big.Int).Add(currentTd, newHeader.Difficulty)
	newHeaderHash := newHeader.Hash()
	if err = hc.bcStore.PutBlockHeader(newHeaderHash, newHeader, newTd, true); err != nil {
		return err
	}

	hc.currentHeaderHash, hc.currentHeader = newHeaderHash, newHeader.Clone()

	return nil
}
