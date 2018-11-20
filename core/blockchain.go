/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/svm"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/metrics"
	leveldbErrors "github.com/syndtr/goleveldb/leveldb/errors"
)

const (
	// limit block should not be ahead of 10 seconds of current time
	futureBlockLimit int64 = 10

	// BlockTransactionNumberLimit block transaction number limit, 1000 simple transactions are about 152kb
	// If for block size as 100KB, it could contains about 5k transactions
	BlockTransactionNumberLimit = 5000

	// BlockByteLimit is the limit of size in bytes
	BlockByteLimit = 1024 * 1024
)

var (
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

	// ErrBlockCreateTimeInFuture is returned when block create time is ahead of 10 seconds of now
	ErrBlockCreateTimeInFuture = errors.New("future block. block time is ahead 10 seconds of now")

	// ErrBlockTooManyTxs is returned when block have too many txs
	ErrBlockTooManyTxs = errors.New("block have too many transactions")

	// ErrBlockExtraDataNotEmpty is returned when the block extra data is not empty.
	ErrBlockExtraDataNotEmpty = errors.New("block extra data is not empty")

	// ErrNotSupported is returned when unsupported method invoked.
	ErrNotSupported = errors.New("not supported function")
)

// Blockchain represents the blockchain with a genesis block. The Blockchain manages
// blocks insertion, deletion, reorganizations and persistence with a given database.
// This is a thread safe structure. we must keep all of its parameters are thread safe too.
type Blockchain struct {
	bcStore        store.BlockchainStore
	accountStateDB database.Database
	engine         consensus.Engine
	genesisBlock   *types.Block
	lock           sync.RWMutex // lock for update blockchain info. for example write block

	blockLeaves  *BlockLeaves
	currentBlock *types.Block
	log          *log.SeeleLog

	rp           *recoveryPoint // used to recover blockchain in case of program crashed when write a block
	debtVerifier types.DebtVerifier
}

// NewBlockchain returns an initialized blockchain with the given store and account state DB.
func NewBlockchain(bcStore store.BlockchainStore, accountStateDB database.Database, recoveryPointFile string, engine consensus.Engine,
	verifier types.DebtVerifier) (*Blockchain, error) {
	bc := &Blockchain{
		bcStore:        bcStore,
		accountStateDB: accountStateDB,
		engine:         engine,
		log:            log.GetLogger("blockchain"),
		debtVerifier:   verifier,
	}

	var err error

	// recover from program crash
	bc.rp, err = loadRecoveryPoint(recoveryPointFile)
	if err != nil {
		bc.log.Error("Failed to load recovery point info from file, %v", err.Error())
		return nil, errors.NewStackedError(err, "failed to load recovery point info")
	}

	if err = bc.rp.recover(bcStore); err != nil {
		bc.log.Error("Failed to recover blockchain, info = %+v, error = %v", *bc.rp, err.Error())
		return nil, errors.NewStackedError(err, "failed to recover blockchain")
	}

	// Get the genesis block from store
	genesisHash, err := bcStore.GetBlockHash(genesisBlockHeight)
	if err != nil {
		bc.log.Error("Failed to get block hash of genesis block height, %v", err.Error())
		return nil, errors.NewStackedErrorf(err, "failed to get genesis block hash by height %v", genesisBlockHeight)
	}

	bc.genesisBlock, err = bcStore.GetBlock(genesisHash)
	if err != nil {
		bc.log.Error("Failed to get block by genesis block hash, hash = %v, error = %v", genesisHash.Hex(), err.Error())
		return nil, errors.NewStackedErrorf(err, "failed to get genesis block by hash %v", genesisHash)
	}

	// Get the HEAD block from store
	currentHeaderHash, err := bcStore.GetHeadBlockHash()
	if err != nil {
		bc.log.Error("Failed to get HEAD block hash, %v", err.Error())
		return nil, errors.NewStackedError(err, "failed to get HEAD block hash")
	}

	if bc.currentBlock, err = bcStore.GetBlock(currentHeaderHash); err != nil {
		bc.log.Error("Failed to get block by HEAD block hash, hash = %v, error = %v", currentHeaderHash.Hex(), err.Error())
		return nil, errors.NewStackedErrorf(err, "failed to get HEAD block by hash %v", currentHeaderHash)
	}

	td, err := bcStore.GetBlockTotalDifficulty(currentHeaderHash)
	if err != nil {
		bc.log.Error("Failed to get HEAD block TD, hash = %v, error = %v", currentHeaderHash.Hex(), err.Error())
		return nil, errors.NewStackedErrorf(err, "failed to get HEAD block TD by hash %v", currentHeaderHash)
	}

	blockIndex := NewBlockIndex(currentHeaderHash, bc.currentBlock.Header.Height, td)
	bc.blockLeaves = NewBlockLeaves()
	bc.blockLeaves.Add(blockIndex)

	return bc, nil
}

// CurrentBlock returns the HEAD block of the blockchain.
func (bc *Blockchain) CurrentBlock() *types.Block {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return bc.currentBlock
}

// CurrentHeader returns the HEAD block header of the blockchain.
func (bc *Blockchain) CurrentHeader() *types.BlockHeader {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	if bc.currentBlock == nil {
		return nil
	}

	return bc.currentBlock.Header
}

// GetCurrentState returns the state DB of the current block.
func (bc *Blockchain) GetCurrentState() (*state.Statedb, error) {
	block := bc.CurrentBlock()
	return state.NewStatedb(block.Header.StateHash, bc.accountStateDB)
}

// GetHeader retrieves a block header from the database by hash and number.
func (bc *Blockchain) GetHeaderByHeight(height uint64) *types.BlockHeader {
	hash, err := bc.bcStore.GetBlockHash(height)
	if err != nil {
		bc.log.Warn("get block header by height failed, err %s. height %d", err, height)
		return nil
	}

	return bc.GetHeaderByHash(hash)
}

// GetHeaderByNumber retrieves a block header from the database by number.
func (bc *Blockchain) GetHeaderByHash(hash common.Hash) *types.BlockHeader {
	header, err := bc.bcStore.GetBlockHeader(hash)
	if err != nil {
		bc.log.Warn("get block header by hash failed, err %s", err)
		return nil
	}

	return header
}

// GetHeaderByHash retrieves a block header from the database by its hash.
func (bc *Blockchain) GetBlockByHash(hash common.Hash) *types.Block {
	block, err := bc.bcStore.GetBlock(hash)
	if err != nil {
		bc.log.Warn("get block by hash failed, err %s", err)
		return nil
	}

	return block
}

// GetState returns the state DB of the specified root hash.
func (bc *Blockchain) GetState(root common.Hash) (*state.Statedb, error) {
	return state.NewStatedb(root, bc.accountStateDB)
}

func (bc *Blockchain) GetStateByRootAndBlockHash(root, blockHash common.Hash) (*state.Statedb, error) {
	panic("unsupported")
}

func (bc *Blockchain) Genesis() *types.Block {
	return bc.genesisBlock
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

// WriteHeader writes the specified head to the blockchain store, only used in lightchain.
func (bc *Blockchain) WriteHeader(*types.BlockHeader) error {
	return ErrNotSupported
}

func (bc *Blockchain) doWriteBlock(block *types.Block) error {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	if err := bc.validateBlock(block); err != nil {
		return errors.NewStackedError(err, "failed to validate block")
	}

	preHeader, err := bc.bcStore.GetBlockHeader(block.Header.PreviousBlockHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to get block header by hash %v", block.Header.PreviousBlockHash)
	}

	// Process the txs in the block and check the state root hash.
	var blockStatedb *state.Statedb
	var receipts []*types.Receipt
	if blockStatedb, receipts, err = bc.applyTxs(block, preHeader.StateHash); err != nil {
		return errors.NewStackedError(err, "failed to apply block txs")
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
		return errors.NewStackedError(err, "failed to commit statedb changes to database batch")
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

	if block.Debts != nil {
		currentBlock.Debts = make([]*types.Debt, len(block.Debts))
		copy(currentBlock.Debts, block.Debts)
	}

	var previousTd *big.Int
	if previousTd, err = bc.bcStore.GetBlockTotalDifficulty(block.Header.PreviousBlockHash); err != nil {
		return errors.NewStackedErrorf(err, "failed to get block TD by hash %v", block.Header.PreviousBlockHash)
	}

	currentTd := new(big.Int).Add(previousTd, block.Header.Difficulty)
	blockIndex := NewBlockIndex(currentBlock.HeaderHash, currentBlock.Header.Height, currentTd)
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
		return errors.NewStackedError(err, "failed to commit statedb changes into database")
	}

	if err = bc.rp.onPutBlockStart(block, bc.bcStore, isHead); err != nil {
		bc.log.Error("Failed to set recovery point before put block into store, %v", err.Error())
		return errors.NewStackedErrorf(err, "failed to set recovery point before put block into store, isNewHead = %v", isHead)
	}

	if err = bc.bcStore.PutReceipts(block.HeaderHash, receipts); err != nil {
		bc.log.Error("Failed to save receipts into store, %v", err.Error())
		return errors.NewStackedErrorf(err, "failed to save receipts into store, blockHash = %v, receipts count = %v", block.HeaderHash, len(receipts))
	}

	if err = bc.bcStore.PutBlock(block, currentTd, isHead); err != nil {
		bc.log.Error("Failed to save block into store, %v", err.Error())
		return errors.NewStackedErrorf(err, "failed to save block into store, blockHash = %v, newTD = %v, isNewHead = %v", block.HeaderHash, currentTd, isHead)
	}

	bc.rp.onPutBlockEnd()

	// If the new block has larger TD, the canonical chain will be changed.
	// In this case, need to update the height-to-blockHash mapping for the new canonical chain.
	if isHead {
		if err = DeleteLargerHeightBlocks(bc.bcStore, block.Header.Height+1, bc.rp); err != nil {
			bc.log.Error("Failed to delete larger height blocks when HEAD changed, larger height = %v, error = %v", block.Header.Height+1, err.Error())
			return errors.NewStackedErrorf(err, "failed to delete larger height blocks, height = %v", block.Header.Height+1)
		}

		if err = OverwriteStaleBlocks(bc.bcStore, block.Header.PreviousBlockHash, bc.rp); err != nil {
			bc.log.Error("Failed to overwrite stale blocks, hash = %v, error = %v", block.Header.PreviousBlockHash, err.Error())
			return errors.NewStackedErrorf(err, "failed to overwrite stale blocks, hash = %v", block.Header.PreviousBlockHash)
		}
	}

	// update block header after meta info updated
	bc.blockLeaves.Add(blockIndex)
	bc.blockLeaves.Remove(block.Header.PreviousBlockHash)

	committed = true
	if isHead {
		bc.currentBlock = currentBlock

		bc.blockLeaves.PurgeAsync(bc.bcStore, func(err error) {
			if err != nil {
				bc.log.Error(errors.NewStackedError(err, "failed to purge block").Error())
			}
		})

		event.ChainHeaderChangedEventMananger.Fire(block.HeaderHash)
	}

	return nil
}

// validateBlock validates all blockhain independent fields in the block.
func (bc *Blockchain) validateBlock(block *types.Block) error {
	if block == nil {
		return types.ErrBlockHeaderNil
	}

	if err := ValidateBlockHeader(block.Header, bc.engine, bc.bcStore, bc); err != nil {
		return errors.NewStackedError(err, "failed to validate block header")
	}

	if err := block.Validate(); err != nil {
		return errors.NewStackedError(err, "failed to validate block")
	}

	if (types.GetTransactionsSize(block.Transactions[1:]) + types.GetDebtsSize(block.Debts)) > BlockByteLimit {
		return ErrBlockTooManyTxs
	}

	// Validate miner shard
	if common.IsShardEnabled() {
		if shard := block.GetShardNumber(); shard != common.LocalShardNumber {
			return fmt.Errorf("invalid shard number. block shard number is [%v], but local shard number is [%v]", shard, common.LocalShardNumber)
		}
	}

	return nil
}

// ValidateBlockHeader validates the specified header.
func ValidateBlockHeader(header *types.BlockHeader, engine consensus.Engine, bcStore store.BlockchainStore, chainReader consensus.ChainReader) error {
	if header == nil {
		return types.ErrBlockHeaderNil
	}

	// Validate timestamp
	if header.CreateTimestamp == nil {
		return ErrBlockCreateTimeNull
	}

	future := new(big.Int).SetInt64(time.Now().Unix() + futureBlockLimit)
	if header.CreateTimestamp.Cmp(future) > 0 {
		return ErrBlockCreateTimeInFuture
	}

	// Now, the extra data in block header should be empty except the genesis block.
	if header.Consensus != types.IstanbulConsensus && len(header.ExtraData) > 0 {
		return ErrBlockExtraDataNotEmpty
	}

	// Do not write the block if already exists.
	blockHash := header.Hash()
	exist, err := bcStore.HasBlock(blockHash)
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to check if block exists by hash %v", blockHash)
	}

	if exist {
		return ErrBlockAlreadyExists
	}

	if err = engine.VerifyHeader(chainReader, header); err != nil {
		return errors.NewStackedError(err, "failed to verify header by consensus engine")
	}

	return nil
}

// GetStore returns the blockchain store instance.
func (bc *Blockchain) GetStore() store.BlockchainStore {
	return bc.bcStore
}

// applyTxs processes the txs in the specified block and returns the new state DB of the block.
// This method supposes the specified block is validated.
func (bc *Blockchain) applyTxs(block *types.Block, root common.Hash) (*state.Statedb, []*types.Receipt, error) {
	minerRewardTx, err := bc.validateMinerRewardTx(block)
	if err != nil {
		return nil, nil, errors.NewStackedError(err, "failed to validate miner reward tx")
	}

	statedb, err := state.NewStatedb(root, bc.accountStateDB)
	if err != nil {
		return nil, nil, errors.NewStackedErrorf(err, "failed to create statedb by root hash %v", root)
	}

	// update debts
	for _, d := range block.Debts {
		_, err = ApplyDebt(statedb, d, block.Header.Creator, bc.debtVerifier)
		if err != nil {
			return nil, nil, errors.NewStackedError(err, "failed to apply debt")
		}
	}

	receipts, err := bc.updateStateDB(statedb, minerRewardTx, block.Transactions[1:], block.Header)
	if err != nil {
		return nil, nil, errors.NewStackedErrorf(err, "failed to update statedb")
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

	reward := consensus.GetReward(block.Header.Height)
	if reward == nil || reward.Cmp(minerRewardTx.Data.Amount) != 0 {
		return nil, fmt.Errorf("invalid reward amount, block height %d, want %s, got %s", block.Header.Height, reward, minerRewardTx.Data.Amount)
	}

	if minerRewardTx.Data.Timestamp != block.Header.CreateTimestamp.Uint64() {
		return nil, types.ErrTimestampMismatch
	}

	return minerRewardTx, nil
}

func (bc *Blockchain) updateStateDB(statedb *state.Statedb, minerRewardTx *types.Transaction, txs []*types.Transaction,
	blockHeader *types.BlockHeader) ([]*types.Receipt, error) {
	// process miner reward
	rewardTxReceipt, err := ApplyRewardTx(minerRewardTx, statedb)
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to apply miner reward tx")
	}

	receipts := make([]*types.Receipt, len(txs)+1)

	// add the receipt of the reward tx
	receipts[0] = rewardTxReceipt

	if err := types.BatchValidateTxs(txs); err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to batch validate %v txs", len(txs))
	}

	// process other txs
	for i, tx := range txs {
		txIdx := i + 1

		if err := tx.ValidateState(statedb); err != nil {
			return nil, errors.NewStackedErrorf(err, "failed to validate tx[%v] against statedb", txIdx)
		}

		receipt, err := bc.ApplyTransaction(tx, txIdx, minerRewardTx.Data.To, statedb, blockHeader)
		if err != nil {
			return nil, errors.NewStackedErrorf(err, "failed to apply tx[%v]", txIdx)
		}

		receipts[txIdx] = receipt
	}

	return receipts, nil
}

// ApplyRewardTx applies a reward transaction, changes corresponding statedb and generates a receipt.
func ApplyRewardTx(rewardTx *types.Transaction, statedb *state.Statedb) (*types.Receipt, error) {
	statedb.CreateAccount(rewardTx.Data.To)
	statedb.AddBalance(rewardTx.Data.To, rewardTx.Data.Amount)

	hash, err := statedb.Hash()
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to get statedb root hash")
	}

	receipt := types.MakeRewardReceipt(rewardTx)
	receipt.PostState = hash

	return receipt, nil
}

// ApplyTransaction applies a transaction, changes corresponding statedb and generates its receipt
func (bc *Blockchain) ApplyTransaction(tx *types.Transaction, txIndex int, coinbase common.Address, statedb *state.Statedb,
	blockHeader *types.BlockHeader) (*types.Receipt, error) {
	ctx := &svm.Context{
		Tx:          tx,
		TxIndex:     txIndex,
		Statedb:     statedb,
		BlockHeader: blockHeader,
		BcStore:     bc.bcStore,
	}
	receipt, err := svm.Process(ctx)
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to process tx via svm")
	}

	return receipt, nil
}

// ApplyDebt applies a debt and update statedb.
func ApplyDebt(statedb *state.Statedb, d *types.Debt, coinbase common.Address, verifier types.DebtVerifier) (recoverable bool, retErr error) {
	data := statedb.GetData(d.Data.Account, d.Hash)
	if bytes.Equal(data, types.DebtDataFlag) {
		retErr = fmt.Errorf("debt already packed, debt hash %s", d.Hash.Hex())
		return
	}

	var err error
	if recoverable, err = d.Validate(verifier, false, common.LocalShardNumber); err != nil {
		retErr = errors.NewStackedError(err, "failed to validate debt")
		return
	}

	if !statedb.Exist(d.Data.Account) {
		statedb.CreateAccount(d.Data.Account)
	}

	// @todo handle contract

	statedb.AddBalance(d.Data.Account, d.Data.Amount)
	statedb.AddBalance(coinbase, d.Fee())
	statedb.SetData(d.Data.Account, d.Hash, types.DebtDataFlag)
	return
}

// DeleteLargerHeightBlocks deletes the height-to-hash mappings with larger height in the canonical chain.
func DeleteLargerHeightBlocks(bcStore store.BlockchainStore, largerHeight uint64, rp *recoveryPoint) error {
	// When recover the blockchain, the larger height block hash may be already deleted before program crash.
	if _, err := deleteCanonicalBlock(bcStore, largerHeight); err != nil {
		return errors.NewStackedErrorf(err, "failed to delete canonical block by height %v", largerHeight)
	}

	for i := largerHeight + 1; ; i++ {
		if rp != nil {
			rp.onDeleteLargerHeightBlocks(i)
		}

		deleted, err := deleteCanonicalBlock(bcStore, i)
		if err != nil {
			return errors.NewStackedErrorf(err, "failed to delete canonical block by height %v", i)
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

// deleteCanonicalBlock deletes the canonical block info for the specified height.
func deleteCanonicalBlock(bcStore store.BlockchainStore, height uint64) (bool, error) {
	hash, err := bcStore.GetBlockHash(height)
	if err == leveldbErrors.ErrNotFound {
		return false, nil
	}

	if err != nil {
		return false, errors.NewStackedErrorf(err, "failed to get block hash by height %v", height)
	}

	// delete the tx/debt indices
	block, err := bcStore.GetBlock(hash)
	if err != nil {
		return false, errors.NewStackedErrorf(err, "failed to get block by hash %v", hash)
	}

	if err = bcStore.DeleteIndices(block); err != nil {
		return false, errors.NewStackedErrorf(err, "failed to delete tx/debt indices of block %v", block.HeaderHash)
	}

	// delete the block hash in canonical chain.
	deleted, err := bcStore.DeleteBlockHash(height)
	if err != nil {
		return false, errors.NewStackedErrorf(err, "failed to delete block hash by height %v", height)
	}

	return deleted, nil
}

// OverwriteStaleBlocks overwrites the stale canonical height-to-hash mappings.
func OverwriteStaleBlocks(bcStore store.BlockchainStore, staleHash common.Hash, rp *recoveryPoint) error {
	var overwritten bool
	var err error

	// When recover the blockchain, the stale block hash my be already overwritten before program crash.
	if _, staleHash, err = overwriteSingleStaleBlock(bcStore, staleHash); err != nil {
		return errors.NewStackedErrorf(err, "failed to overwrite single stale block, hash = %v", staleHash)
	}

	for !staleHash.Equal(common.EmptyHash) {
		if rp != nil {
			rp.onOverwriteStaleBlocks(staleHash)
		}

		if overwritten, staleHash, err = overwriteSingleStaleBlock(bcStore, staleHash); err != nil {
			return errors.NewStackedErrorf(err, "failed to overwrite single stale block, hash = %v", staleHash)
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
		return false, common.EmptyHash, errors.NewStackedErrorf(err, "failed to get block header by hash %v", hash)
	}

	canonicalHash, err := bcStore.GetBlockHash(header.Height)
	if err != nil {
		return false, common.EmptyHash, errors.NewStackedErrorf(err, "failed to get block hash by height %v in canonical chain", header.Height)
	}

	if hash.Equal(canonicalHash) {
		return false, header.PreviousBlockHash, nil
	}

	// delete the tx/debt indices in previous canonical chain.
	canonicalBlock, err := bcStore.GetBlock(canonicalHash)
	if err != nil {
		return false, common.EmptyHash, errors.NewStackedErrorf(err, "failed to get block by hash %v", canonicalHash)
	}

	if err = bcStore.DeleteIndices(canonicalBlock); err != nil {
		return false, common.EmptyHash, errors.NewStackedErrorf(err, "failed to delete tx/debt indices of block %v", canonicalBlock.HeaderHash)
	}

	// add the tx/debt indices in new canonical chain.
	block, err := bcStore.GetBlock(hash)
	if err != nil {
		return false, common.EmptyHash, errors.NewStackedErrorf(err, "failed to get block by hash %v", hash)
	}

	if err = bcStore.AddIndices(block); err != nil {
		return false, common.EmptyHash, errors.NewStackedErrorf(err, "failed to add tx/debt indices of block %v", block.HeaderHash)
	}

	// update the block hash in canonical chain.
	if err = bcStore.PutBlockHash(header.Height, hash); err != nil {
		return false, common.EmptyHash, errors.NewStackedErrorf(err, "failed to put block height to hash map in canonical chain, height = %v, hash = %v", header.Height, hash)
	}

	return true, header.PreviousBlockHash, nil
}

// GetShardNumber returns the shard number of blockchain.
func (bc *Blockchain) GetShardNumber() (uint, error) {
	data, err := getShardInfo(bc.genesisBlock)
	if err != nil {
		return 0, errors.NewStackedError(err, "failed to get extra data in genesis block")
	}

	return data.ShardNumber, nil
}
