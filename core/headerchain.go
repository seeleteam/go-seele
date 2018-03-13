/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
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
