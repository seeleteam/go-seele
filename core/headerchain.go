/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

// HeaderChain represents the block header chain that is shared by the archive node and light node.
// This is a non-thread safe structure.
type HeaderChain struct {
	lock    sync.Mutex
	bcStore store.BlockchainStore

	genesisHeader     *types.BlockHeader
	currentHeader     *types.BlockHeader
	currentHeaderHash common.Hash
}

// NewHeaderChain returns a new instance of the HeaderChain structure.
func NewHeaderChain(bcStore store.BlockchainStore) (*HeaderChain, error) {
	hc := HeaderChain{
		bcStore: bcStore,
	}

	// Get the genesis block header from the store.
	genesisHash, err := bcStore.GetBlockHash(genesisBlockHeight)
	if err != nil {
		return nil, err
	}

	hc.genesisHeader, err = bcStore.GetBlockHeader(genesisHash)
	if err != nil {
		return nil, err
	}

	// Get the HEAD block header from the store.
	hc.currentHeaderHash, err = bcStore.GetHeadBlockHash()
	if err != nil {
		return nil, err
	}

	hc.currentHeader, err = bcStore.GetBlockHeader(hc.currentHeaderHash)
	if err != nil {
		return nil, err
	}

	hc.lock = sync.Mutex{}

	return &hc, nil
}

// WriteHeader writes a new block header into the header chain.
// It requires the new header's parent header is the HEAD header
// in the chain.
func (hc *HeaderChain) WriteHeader(newHeader *types.BlockHeader) error {
	hc.lock.Lock()
	defer hc.lock.Unlock()

	newHeaderHash := newHeader.Hash()
	hc.currentHeaderHash, hc.currentHeader = newHeaderHash, newHeader.Clone()

	return nil
}
