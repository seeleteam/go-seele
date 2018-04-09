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

var (
	// ErrBlockHashMismatch is returned when block hash does not match the header hash.
	ErrBlockHashMismatch = errors.New("block header hash mismatch")

	// ErrBlockTxsHashMismatch is returned when block transations hash does not match
	// the transaction root hash in header.
	ErrBlockTxsHashMismatch = errors.New("block transactions root hash mismatch")

	// ErrBlockInvalidParentHash is returned when insert a new header with invalid parent block hash.
	ErrBlockInvalidParentHash = errors.New("invalid parent block hash")

	// ErrBlockInvalidHeight is returned when insert a new header with invalid block height.
	ErrBlockInvalidHeight = errors.New("invalid block height")

	// ErrBlockAlreadyExist is returned when inserted block already exist
	ErrBlockAlreadyExist = errors.New("block already exist")

	// ErrBlockStateHashInvalid is returned when block state hash is invalid
	ErrBlockStateHashInvalid = errors.New("block state root hash is invalid")
)

// Blockchain represents the block chain with a genesis block. The Blockchain manages
// blocks insertion, deletion, reorganizations and persistence with a given database.
// This is a thread safe structure. we must keep all of its parameter are thread safe too.
type Blockchain struct {
	bcStore        store.BlockchainStore
	accountStateDB database.Database
	headerChain    *HeaderChain
	genesisBlock   *types.Block
	lock           sync.Mutex // lock for update blockchain info. for example write block

	blockLeaves *BlockLeaves
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

	currentBlock, err := bcStore.GetBlock(currentHeaderHash)
	if err != nil {
		return nil, err
	}

	td, err := bcStore.GetBlockTotalDifficulty(currentHeaderHash)
	if err != nil {
		return nil, err
	}

	// Get the state DB of current block
	currentState, err := state.NewStatedb(currentBlock.Header.StateHash, accountStateDB)
	if err != nil {
		return nil, err
	}

	blockIndex := NewBlockIndex(currentState, currentBlock, td)
	bc.blockLeaves = NewBlockLeaf()
	bc.blockLeaves.Add(blockIndex)

	return bc, nil
}

// CurrentBlock returns the HEAD block of blockchain.
func (bc *Blockchain) CurrentBlock() (*types.Block, *state.Statedb) {
	index := bc.blockLeaves.GetBestBlockIndex()
	if index == nil {
		return nil, nil
	}

	return index.currentBlock, index.state
}

func (bc *Blockchain) CurrentState() *state.Statedb {
	_, state := bc.CurrentBlock()
	return state
}

// WriteBlock writes the block to the blockchain store.
func (bc *Blockchain) WriteBlock(block *types.Block) error {
	exist, _ := bc.bcStore.HashBlock(block.HeaderHash)
	if exist {
		return ErrBlockAlreadyExist
	}

	parent, err := bc.bcStore.GetBlock(block.Header.PreviousBlockHash)
	if err != nil {
		return ErrBlockInvalidParentHash
	}

	err = bc.ValidateBlock(block, parent)
	if err != nil {
		return err
	}

	blockStatedb, err := bc.getUpdatedState(block, parent)
	if err != nil {
		return err
	}

	currentBlock := &types.Block{
		HeaderHash:   block.HeaderHash,
		Header:       block.Header.Clone(),
		Transactions: make([]*types.Transaction, len(block.Transactions)),
	}
	copy(currentBlock.Transactions, block.Transactions)

	td, err := bc.bcStore.GetBlockTotalDifficulty(block.Header.PreviousBlockHash)
	if err != nil {
		return err
	}

	blockIndex := NewBlockIndex(blockStatedb, currentBlock, td.Add(td, block.Header.Difficulty))

	bc.lock.Lock()
	defer bc.lock.Unlock()
	isHead := bc.blockLeaves.IsBestBlockIndex(blockIndex)
	bc.blockLeaves.Add(blockIndex)
	bc.blockLeaves.RemoveByHash(block.Header.PreviousBlockHash)
	bc.headerChain.WriteHeader(currentBlock.Header)

	err = bc.bcStore.PutBlock(block, td, isHead)
	if err != nil {
		return err
	}

	return nil
}

func (bc *Blockchain) getUpdatedState(block *types.Block, parent *types.Block) (*state.Statedb, error) {
	blockStatedb, err := state.NewStatedb(parent.Header.StateHash, bc.accountStateDB)
	if err != nil {
		return nil, err
	}

	for index, tx := range block.Transactions {
		// block miner reward transaction
		if index == 0 {
			blockStatedb.AddAmount(*tx.Data.To, tx.Data.Amount)
			continue
		}

		blockStatedb.SubAmount(tx.Data.From, tx.Data.Amount)
		blockStatedb.AddAmount(*tx.Data.To, tx.Data.Amount)
	}

	batch := bc.accountStateDB.NewBatch()
	root, err := blockStatedb.Commit(batch)
	if err != nil {
		return nil, err
	}

	err = batch.Commit()
	if err != nil {
		return nil, err
	}

	if root != block.Header.StateHash {
		return nil, ErrBlockStateHashInvalid
	}

	return blockStatedb, nil
}

func (bc *Blockchain) ValidateBlock(block *types.Block, parent *types.Block) error {
	if !block.HeaderHash.Equal(block.Header.Hash()) {
		return ErrBlockHashMismatch
	}

	txsHash := types.MerkleRootHash(block.Transactions)
	if !txsHash.Equal(block.Header.TxHash) {
		return ErrBlockTxsHashMismatch
	}

	// @todo transaction validation

	if block.Header.Height != parent.Header.Height+1 {
		return ErrBlockInvalidHeight
	}

	return nil
}

func (bc *Blockchain) GetStore() store.BlockchainStore {
	return bc.bcStore
}
