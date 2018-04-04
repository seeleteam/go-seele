/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"errors"
	"sync"

	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
)

// ErrBlockHashMismatch is returned when block hash does not match the header hash.
var ErrBlockHashMismatch = errors.New("block header hash mismatch")

// ErrBlockTxsHashMismatch is returned when block transations hash does not match
// the transaction root hash in header.
var ErrBlockTxsHashMismatch = errors.New("block transactions root hash mismatch")

// Blockchain represents the block chain with a genesis block. The Blockchain manages
// blocks insertion, deletion, reorganizations and persistence with a given database.
// This is a thread safe structure.
type Blockchain struct {
	mutex          sync.RWMutex
	bcStore        store.BlockchainStore
	accountStateDB database.Database
	headerChain    *HeaderChain

	genesisBlock *types.Block
	currentBlock *types.Block

	currentState *state.Statedb
}

// NewBlockchain returns a initialized block chain with given store and account state DB.
func NewBlockchain(bcStore store.BlockchainStore, accountStateDB database.Database) (*Blockchain, error) {
	bc := &Blockchain{
		bcStore:        bcStore,
		accountStateDB: accountStateDB,
	}

	var err error
	bc.headerChain, err = NewHeaderChain(bcStore)
	if err != nil {
		return nil, err
	}

	// Get genesis block from store
	genesisHash, err := bcStore.GetBlockHash(genesisBlockHeight)
	if err != nil {
		return nil, err
	}

	bc.genesisBlock, err = bcStore.GetBlock(genesisHash)
	if err != nil {
		return nil, err
	}

	// Get HEAD block from store
	currentHeaderHash, err := bcStore.GetHeadBlockHash()
	if err != nil {
		return nil, err
	}

	bc.currentBlock, err = bcStore.GetBlock(currentHeaderHash)
	if err != nil {
		return nil, err
	}

	// Get the state DB of current block
	bc.currentState, err = state.NewStatedb(bc.currentBlock.Header.StateHash, accountStateDB)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// CurrentBlock returns the HEAD block of blockchain.
func (bc *Blockchain) CurrentBlock() *types.Block {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	return bc.currentBlock
}

// WriteBlock writes the block to the blockchain store.
func (bc *Blockchain) WriteBlock(block *types.Block) error {
	if !block.HeaderHash.Equal(block.Header.Hash()) {
		return ErrBlockHashMismatch
	}

	txsHash := types.MerkleRootHash(block.Transactions)
	if !txsHash.Equal(block.Header.TxHash) {
		return ErrBlockTxsHashMismatch
	}

	blockStatedb, err := state.NewStatedb(block.Header.StateHash, bc.accountStateDB)
	if err != nil {
		return err
	}

	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	newTd, err := bc.headerChain.validateNewHeader(block.Header)
	if err != nil {
		return err
	}

	err = bc.bcStore.PutBlock(block, newTd, true)
	if err != nil {
		return err
	}

	bc.currentBlock = &types.Block{
		HeaderHash:   block.HeaderHash,
		Header:       block.Header.Clone(),
		Transactions: make([]*types.Transaction, len(block.Transactions)),
	}

	copy(bc.currentBlock.Transactions, block.Transactions)

	bc.headerChain.currentHeaderHash = bc.currentBlock.HeaderHash
	bc.headerChain.currentHeader = bc.currentBlock.Header
	bc.currentState = blockStatedb

	return nil
}

// GetStore returns the blockchain store instance.
func (bc *Blockchain) GetStore() store.BlockchainStore {
	return bc.bcStore
}

// CurrentState returns the state DB of current block.
func (bc *Blockchain) CurrentState() *state.Statedb {
	bc.mutex.RLock()
	defer bc.mutex.RUnlock()

	return bc.currentState
}
