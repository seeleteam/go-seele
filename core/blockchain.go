/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"github.com/seeleteam/go-seele/core/store"
)

// Blockchain represents the block chain with a genesis block. The Blockchain manages
// blocks insertion, deletion, reorganizations and persistence with a given database.
// This is a thread safe structure.
type Blockchain struct {
	bcStore     store.BlockchainStore
	headerChain *HeaderChain
}

// NewBlockchain returns a initialized block chain with given store.
func NewBlockchain(bcStore store.BlockchainStore) (*Blockchain, error) {
	bc := &Blockchain{
		bcStore: bcStore,
	}

	var err error
	bc.headerChain, err = NewHeaderChain(bcStore)
	if err != nil {
		return nil, err
	}

	return bc, nil
}
