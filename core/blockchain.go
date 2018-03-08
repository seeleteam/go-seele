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
type Blockchain struct {
	bcStore store.BlockchainStore
}

// NewBlockchain returns a initialized block chain with given store.
func NewBlockchain(bcStore store.BlockchainStore) *Blockchain {
	return &Blockchain{bcStore}
}
