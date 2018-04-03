/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"errors"
	"sync"

	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
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
	mutex       sync.RWMutex
	bcStore     store.BlockchainStore
	headerChain *HeaderChain

	genesisBlock *types.Block
	currentBlock *types.Block
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

	return nil
}

func (bc *Blockchain) GetBlockChainStore() store.BlockchainStore {
	return bc.bcStore
}
