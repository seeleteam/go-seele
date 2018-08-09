/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/core/vm"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/metrics"
	"github.com/seeleteam/go-seele/miner/pow"
)

const (
	// limit block should not be ahead of 10 seconds of current time
	futureBlockLimit int64 = 10

	// block transaction number limit, 1000 simple transactions are about 152kb
	// If for block size as 100KB, it could contains about 5k transactions
	BlockTransactionNumberLimit = 5000

	// BlockByteLimit is the limit of size in bytes
	BlockByteLimit = 1024 * 1024
)

var (
	// ErrBlockHeaderNil is returned when the block header is nil.
	ErrBlockHeaderNil = errors.New("block header is nil")

	// ErrBlockHashMismatch is returned when the block hash does not match the header hash.
	ErrBlockHashMismatch = errors.New("block header hash mismatch")

	// ErrBlockTxsHashMismatch is returned when the block transactions hash does not match
	// the transaction root hash in the header.
	ErrBlockTxsHashMismatch = errors.New("block transactions root hash mismatch")

	// ErrBlockInvalidParentHash is returned when inserting a new header with invalid parent block hash.
	ErrBlockInvalidParentHash = errors.New("invalid parent block hash")

	// ErrBlockInvalidHeight is returned when inserting a new header with invalid block height.
	ErrBlockInvalidHeight = errors.New("invalid block height")

	// ErrBlockAlreadyExists is returned when inserted block already exists
	ErrBlockAlreadyExists = errors.New("block already exists")

	// ErrBlockStateHashMismatch is returned when the calculated account state hash of block
	// does not match the state root hash in block header.
	ErrBlockStateHashMismatch = errors.New("block state hash mismatch")

	// ErrBlockReceiptHashMismatch is returned when the calculated receipts hash of block
	// does not match the receipts root hash in block header.
	ErrBlockReceiptHashMismatch = errors.New("block receipts hash mismatch")

	// ErrBlockEmptyTxs is returned when writing a block with empty transactions.
	ErrBlockEmptyTxs = errors.New("empty transactions in block")

	// ErrBlockInvalidToAddress is returned when the to address of miner reward tx is nil.
	ErrBlockInvalidToAddress = errors.New("invalid to address")

	// ErrBlockCoinbaseMismatch is returned when the to address of miner reward tx does not match
	// the creator address in the block header.
	ErrBlockCoinbaseMismatch = errors.New("coinbase mismatch")

	// ErrBlockCreateTimeNull is returned when block create time is nil
	ErrBlockCreateTimeNull = errors.New("block must have create time")

	// ErrBlockCreateTimeOld is returned when block create time is previous of parent block time
	ErrBlockCreateTimeOld = errors.New("block time must be later than parent block time")

	// ErrBlockCreateTimeInFuture is returned when block create time is ahead of 10 seconds of now
	ErrBlockCreateTimeInFuture = errors.New("future block. block time is ahead 10 seconds of now")

	// ErrBlockDifficultInvalid is returned when block difficult is invalid
	ErrBlockDifficultInvalid = errors.New("block difficult is invalid")

	// ErrBlockTooManyTxs is returned when block have too many txs
	ErrBlockTooManyTxs = errors.New("block have too many transactions")

	// ErrBlockExtraDataNotEmpty is returned when the block extra data is not empty.
	ErrBlockExtraDataNotEmpty = errors.New("block extra data is not empty")
)

type consensusEngine interface {
	// ValidateHeader validates the specified header and return error if validation failed.
	// Generally, need to validate the block nonce.
	ValidateHeader(blockHeader *types.BlockHeader) error

	// ValidateRewardAmount validates the specified amount and returns error if validation failed.
	// The amount of miner reward will change over time.
	ValidateRewardAmount(blockHeight uint64, amount *big.Int) error
}

// Blockchain represents the blockchain with a genesis block. The Blockchain manages
// blocks insertion, deletion, reorganizations and persistence with a given database.
// This is a thread safe structure. we must keep all of its parameters are thread safe too.
type Blockchain struct {
	bcStore        store.BlockchainStore
	accountStateDB database.Database
	engine         consensusEngine
	genesisBlock   *types.Block
	lock           sync.RWMutex // lock for update blockchain info. for example write block

	blockLeaves *BlockLeaves
	log         *log.SeeleLog

	rp *recoveryPoint // used to recover blockchain in case of program crashed when write a block
}

// NewBlockchain returns an initialized blockchain with the given store and account state DB.
func NewBlockchain(bcStore store.BlockchainStore, accountStateDB database.Database, recoveryPointFile string) (*Blockchain, error) {
	bc := &Blockchain{
		bcStore:        bcStore,
		accountStateDB: accountStateDB,
		engine:         &pow.Engine{},
		log:            log.GetLogger("blockchain"),
	}

	var err error

	// recover from program crash
	bc.rp, err = loadRecoveryPoint(recoveryPointFile)
	if err != nil {
		bc.log.Error("Failed to load recovery point info from file, %v", err.Error())
		return nil, err
	}

	if err = bc.rp.recover(bcStore); err != nil {
		bc.log.Error("Failed to recover blockchain, info = %+v, error = %v", *bc.rp, err.Error())
		return nil, err
	}

	// Get the genesis block from store
	genesisHash, err := bcStore.GetBlockHash(genesisBlockHeight)
	if err != nil {
		bc.log.Error("Failed to get block hash of genesis block height, %v", err.Error())
		return nil, err
	}

	bc.genesisBlock, err = bcStore.GetBlock(genesisHash)
	if err != nil {
		bc.log.Error("Failed to get block by genesis block hash, hash = %v, error = %v", genesisHash.ToHex(), err.Error())
		return nil, err
	}

	// Get the HEAD block from store
	currentHeaderHash, err := bcStore.GetHeadBlockHash()
	if err != nil {
		bc.log.Error("Failed to get HEAD block hash, %v", err.Error())
		return nil, err
	}

	currentBlock, err := bcStore.GetBlock(currentHeaderHash)
	if err != nil {
		bc.log.Error("Failed to get block by HEAD block hash, hash = %v, error = %v", currentHeaderHash.ToHex(), err.Error())
		return nil, err
	}

	td, err := bcStore.GetBlockTotalDifficulty(currentHeaderHash)
	if err != nil {
		bc.log.Error("Failed to get HEAD block TD, hash = %v, error = %v", currentHeaderHash.ToHex(), err.Error())
		return nil, err
	}

	// Get the state DB of the current block
	currentState, err := state.NewStatedb(currentBlock.Header.StateHash, accountStateDB)
	if err != nil {
		bc.log.Error("Failed to create state DB, state hash = %v, error = %v", currentBlock.Header.StateHash, err.Error())
		return nil, err
	}

	blockIndex := NewBlockIndex(currentState, currentBlock, td)
	bc.blockLeaves = NewBlockLeaves()
	bc.blockLeaves.Add(blockIndex)

	return bc, nil
}

// CurrentBlock returns the HEAD block of the blockchain.
func (bc *Blockchain) CurrentBlock() *types.Block {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	index := bc.blockLeaves.GetBestBlockIndex()
	if index == nil {
		return nil
	}

	return index.currentBlock
}

// GetCurrentState returns the state DB of the current block.
func (bc *Blockchain) GetCurrentState() (*state.Statedb, error) {
	block := bc.CurrentBlock()
	return state.NewStatedb(block.Header.StateHash, bc.accountStateDB)
}

// GetCurrentInfo return the current block and current state info
func (bc *Blockchain) GetCurrentInfo() (*types.Block, *state.Statedb, error) {
	block := bc.CurrentBlock()
	statedb, err := state.NewStatedb(block.Header.StateHash, bc.accountStateDB)
	return block, statedb, err
}

// WriteBlock writes the specified block to the blockchain store.
func (bc *Blockchain) WriteBlock(block *types.Block) error {
	startWriteBlockTime := time.Now()
	if err := bc.doWriteBlock(block); err != nil {
		return err
	}
	markTime := time.Since(startWriteBlockTime)
	metrics.MetricsWriteBlockMeter.Mark(markTime.Nanoseconds())
	return nil
}

func (bc *Blockchain) doWriteBlock(block *types.Block) error {
	if err := bc.validateBlock(block); err != nil {
		return err
	}

	// Do not write the block if already exists.
	exist, err := bc.bcStore.HasBlock(block.HeaderHash)
	if err != nil {
		return err
	}

	if exist {
		return ErrBlockAlreadyExists
	}

	bc.lock.Lock()
	defer bc.lock.Unlock()

	var preBlock *types.Block
	if preBlock, err = bc.bcStore.GetBlock(block.Header.PreviousBlockHash); err != nil {
		return ErrBlockInvalidParentHash
	}

	// Ensure the specified block is valid to insert.
	if err = bc.validateBlockInChain(block, preBlock); err != nil {
		return err
	}

	// Process the txs in the block and check the state root hash.
	var blockStatedb *state.Statedb
	var receipts []*types.Receipt
	if blockStatedb, receipts, err = bc.applyTxs(block, preBlock); err != nil {
		return err
	}

	// Validate receipts root hash.
	if receiptsRootHash := types.ReceiptMerkleRootHash(receipts); !receiptsRootHash.Equal(block.Header.ReceiptHash) {
		return ErrBlockReceiptHashMismatch
	}

	// Validate state root hash.
	batch := bc.accountStateDB.NewBatch()
	committed := false
	defer func() {
		if !committed {
			batch.Rollback()
		}
	}()

	var stateRootHash common.Hash
	if stateRootHash, err = blockStatedb.Commit(batch); err != nil {
		return err
	}

	if !stateRootHash.Equal(block.Header.StateHash) {
		return ErrBlockStateHashMismatch
	}

	// Update block leaves and write the block into store.
	currentBlock := &types.Block{
		HeaderHash:   block.HeaderHash,
		Header:       block.Header.Clone(),
		Transactions: make([]*types.Transaction, len(block.Transactions)),
	}
	copy(currentBlock.Transactions, block.Transactions)

	var previousTd *big.Int
	if previousTd, err = bc.bcStore.GetBlockTotalDifficulty(block.Header.PreviousBlockHash); err != nil {
		return err
	}

	currentTd := new(big.Int).Add(previousTd, block.Header.Difficulty)
	blockIndex := NewBlockIndex(blockStatedb, currentBlock, currentTd)
	isHead := bc.blockLeaves.IsBestBlockIndex(blockIndex)

	/////////////////////////////////////////////////////////////////
	// PAY ATTENTION TO THE ORDER OF WRITING DATA INTO DB.
	// OTHERWISE, THERE MAY BE INCONSISTENT DATA.
	// 1. Write account states
	// 2. Write receipts
	// 3. Write block
	/////////////////////////////////////////////////////////////////
	if err = batch.Commit(); err != nil {
		bc.log.Error("Failed to batch commit account states, %v", err.Error())
		return err
	}

	if err = bc.rp.onPutBlockStart(block, bc.bcStore, isHead); err != nil {
		bc.log.Error("Failed to set recovery point before put block into store, %v", err.Error())
		return err
	}

	if err = bc.bcStore.PutReceipts(block.HeaderHash, receipts); err != nil {
		bc.log.Error("Failed to save receipts into store, %v", err.Error())
		return err
	}

	if err = bc.bcStore.PutBlock(block, currentTd, isHead); err != nil {
		bc.log.Error("Failed to save block into store, %v", err.Error())
		return err
	}

	bc.rp.onPutBlockEnd()

	// If the new block has larger TD, the canonical chain will be changed.
	// In this case, need to update the height-to-blockHash mapping for the new canonical chain.
	if isHead {
		if err = deleteLargerHeightBlocks(bc.bcStore, block.Header.Height+1, bc.rp); err != nil {
			bc.log.Error("Failed to delete larger height blocks when HEAD changed, larger height = %v, error = %v", block.Header.Height+1, err.Error())
			return err
		}

		if err = overwriteStaleBlocks(bc.bcStore, block.Header.PreviousBlockHash, bc.rp); err != nil {
			bc.log.Error("Failed to overwrite stale blocks, hash = %v, error = %v", block.Header.PreviousBlockHash, err.Error())
			return err
		}
	}

	// update block header after meta info updated
	bc.blockLeaves.Add(blockIndex)
	bc.blockLeaves.RemoveByHash(block.Header.PreviousBlockHash)

	committed = true
	if isHead {
		event.ChainHeaderChangedEventMananger.Fire(block.HeaderHash)
	}

	return nil
}

// validateBlock validates all blockhain independent fields in the block.
func (bc *Blockchain) validateBlock(block *types.Block) error {
	if block == nil || block.Header == nil {
		return ErrBlockHeaderNil
	}

	if !block.HeaderHash.Equal(block.Header.Hash()) {
		return ErrBlockHashMismatch
	}

	if types.GetTransactionsSize(block.Transactions) > BlockByteLimit {
		return ErrBlockTooManyTxs
	}

	// Validate timestamp
	if block.Header.CreateTimestamp == nil {
		return ErrBlockCreateTimeNull
	}

	future := new(big.Int).SetInt64(time.Now().Unix() + futureBlockLimit)
	if block.Header.CreateTimestamp.Cmp(future) > 0 {
		return ErrBlockCreateTimeInFuture
	}

	// Now, the extra data in block header should be empty except the genesis block.
	if len(block.Header.ExtraData) > 0 {
		return ErrBlockExtraDataNotEmpty
	}

	// Validate miner shard
	if common.IsShardEnabled() {
		if shard := block.GetShardNumber(); shard != common.LocalShardNumber {
			return fmt.Errorf("invalid shard number. block shard number is [%v], but local shard number is [%v]", shard, common.LocalShardNumber)
		}
	}

	// Validate tx merkle root hash
	txsHash := types.MerkleRootHash(block.Transactions)
	if !txsHash.Equal(block.Header.TxHash) {
		return ErrBlockTxsHashMismatch
	}

	return bc.engine.ValidateHeader(block.Header)
}

// validateBlockInChain validates the specified block against with the previous block.
func (bc *Blockchain) validateBlockInChain(block, preBlock *types.Block) error {
	if block.Header.Height != preBlock.Header.Height+1 {
		return ErrBlockInvalidHeight
	}

	if block.Header.CreateTimestamp.Cmp(preBlock.Header.CreateTimestamp) < 0 {
		return ErrBlockCreateTimeOld
	}

	difficult := pow.GetDifficult(block.Header.CreateTimestamp.Uint64(), preBlock.Header)
	if difficult == nil || difficult.Cmp(block.Header.Difficulty) != 0 {
		return ErrBlockDifficultInvalid
	}

	return nil
}

// GetStore returns the blockchain store instance.
func (bc *Blockchain) GetStore() store.BlockchainStore {
	return bc.bcStore
}

// applyTxs processes the txs in the specified block and returns the new state DB of the block.
// This method supposes the specified block is validated.
func (bc *Blockchain) applyTxs(block, preBlock *types.Block) (*state.Statedb, []*types.Receipt, error) {
	minerRewardTx, err := bc.validateMinerRewardTx(block)
	if err != nil {
		return nil, nil, err
	}

	statedb, err := state.NewStatedb(preBlock.Header.StateHash, bc.accountStateDB)
	if err != nil {
		return nil, nil, err
	}

	receipts, err := bc.updateStateDB(statedb, minerRewardTx, block.Transactions[1:], block.Header)
	if err != nil {
		return nil, nil, err
	}

	return statedb, receipts, nil
}

func (bc *Blockchain) validateMinerRewardTx(block *types.Block) (*types.Transaction, error) {
	if len(block.Transactions) == 0 {
		return nil, ErrBlockEmptyTxs
	}

	minerRewardTx := block.Transactions[0]
	if minerRewardTx.Data.To.IsEmpty() {
		return nil, ErrBlockInvalidToAddress
	}

	if !bytes.Equal(minerRewardTx.Data.To.Bytes(), block.Header.Creator.Bytes()) {
		return nil, ErrBlockCoinbaseMismatch
	}

	if minerRewardTx.Data.Amount == nil {
		return nil, types.ErrAmountNil
	}

	if minerRewardTx.Data.Amount.Sign() < 0 {
		return nil, types.ErrAmountNegative
	}

	if err := bc.engine.ValidateRewardAmount(block.Header.Height, minerRewardTx.Data.Amount); err != nil {
		return nil, err
	}

	if minerRewardTx.Data.Timestamp != block.Header.CreateTimestamp.Uint64() {
		return nil, types.ErrTimestampMismatch
	}

	return minerRewardTx, nil
}

func (bc *Blockchain) updateStateDB(statedb *state.Statedb, minerRewardTx *types.Transaction, txs []*types.Transaction, blockHeader *types.BlockHeader) ([]*types.Receipt, error) {
	// process miner reward
	rewardTxReceipt, err := ApplyRewardTx(minerRewardTx, statedb)
	if err != nil {
		return nil, err
	}

	receipts := make([]*types.Receipt, len(txs)+1)

	// add the receipt of the reward tx
	receipts[0] = rewardTxReceipt

	if err := types.BatchValidateTxs(txs); err != nil {
		return nil, err
	}

	// process other txs
	for i, tx := range txs {
		if err := tx.ValidateState(statedb); err != nil {
			return nil, err
		}

		receipt, err := bc.ApplyTransaction(tx, i+1, minerRewardTx.Data.To, statedb, blockHeader)
		if err != nil {
			return nil, err
		}

		receipts[i+1] = receipt
	}

	return receipts, nil
}

// ApplyRewardTx applies a reward transaction, changes corresponding statedb and generates a receipt.
func ApplyRewardTx(rewardTx *types.Transaction, statedb *state.Statedb) (*types.Receipt, error) {
	statedb.CreateAccount(rewardTx.Data.To)
	statedb.AddBalance(rewardTx.Data.To, rewardTx.Data.Amount)

	hash, err := statedb.Hash()
	if err != nil {
		return nil, err
	}

	receipt := types.MakeRewardReceipt(rewardTx)
	receipt.PostState = hash

	return receipt, nil
}

// ApplyTransaction applies a transaction, changes corresponding statedb and generates its receipt
func (bc *Blockchain) ApplyTransaction(tx *types.Transaction, txIndex int, coinbase common.Address, statedb *state.Statedb, blockHeader *types.BlockHeader) (*types.Receipt, error) {
	context := NewEVMContext(tx, blockHeader, coinbase, bc.bcStore)
	receipt, err := ProcessContract(context, tx, txIndex, statedb, &vm.Config{})
	if err != nil {
		return nil, err
	}

	return receipt, nil
}

// deleteLargerHeightBlocks deletes the height-to-hash mappings with larger height in the canonical chain.
func deleteLargerHeightBlocks(bcStore store.BlockchainStore, largerHeight uint64, rp *recoveryPoint) error {
	// When recover the blockchain, the larger height block hash may be already deleted before program crash.
	if _, err := bcStore.DeleteBlockHash(largerHeight); err != nil {
		return err
	}

	for i := largerHeight + 1; ; i++ {
		if rp != nil {
			rp.onDeleteLargerHeightBlocks(i)
		}

		deleted, err := bcStore.DeleteBlockHash(i)
		if err != nil {
			return err
		}

		if !deleted {
			break
		}
	}

	if rp != nil {
		rp.onDeleteLargerHeightBlocks(0)
	}

	return nil
}

// overwriteStaleBlocks overwrites the stale canonical height-to-hash mappings.
func overwriteStaleBlocks(bcStore store.BlockchainStore, staleHash common.Hash, rp *recoveryPoint) error {
	var overwritten bool
	var err error

	// When recover the blockchain, the stale block hash my be already overwritten before program crash.
	if _, staleHash, err = overwriteSingleStaleBlock(bcStore, staleHash); err != nil {
		return err
	}

	for !staleHash.Equal(common.EmptyHash) {
		if rp != nil {
			rp.onOverwriteStaleBlocks(staleHash)
		}

		if overwritten, staleHash, err = overwriteSingleStaleBlock(bcStore, staleHash); err != nil {
			return err
		}

		if !overwritten {
			break
		}
	}

	if rp != nil {
		rp.onOverwriteStaleBlocks(common.EmptyHash)
	}

	return nil
}

func overwriteSingleStaleBlock(bcStore store.BlockchainStore, hash common.Hash) (overwritten bool, preBlockHash common.Hash, err error) {
	header, err := bcStore.GetBlockHeader(hash)
	if err != nil {
		return false, common.EmptyHash, err
	}

	canonicalHash, err := bcStore.GetBlockHash(header.Height)
	if err != nil {
		return false, common.EmptyHash, err
	}

	if hash.Equal(canonicalHash) {
		return false, header.PreviousBlockHash, nil
	}

	if err = bcStore.PutBlockHash(header.Height, hash); err != nil {
		return false, common.EmptyHash, err
	}

	return true, header.PreviousBlockHash, nil
}

// GetShardNumber returns the shard number of blockchian.
func (bc *Blockchain) GetShardNumber() (uint, error) {
	data, err := getGenesisExtraData(bc.genesisBlock)
	if err != nil {
		return 0, err
	}

	return data.ShardNumber, nil
}
