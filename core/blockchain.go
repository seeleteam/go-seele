/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/svm"
	"github.com/seeleteam/go-seele/core/txs"
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
	currentBlock atomic.Value
	log          *log.SeeleLog

	rp           *recoveryPoint // used to recover blockchain in case of program crashed when write a block
	debtVerifier types.DebtVerifier

	lastBlockTime time.Time // last sucessful written block time.
}

// NewBlockchain returns an initialized blockchain with the given store and account state DB.
func NewBlockchain(bcStore store.BlockchainStore, accountStateDB database.Database, recoveryPointFile string, engine consensus.Engine,
	verifier types.DebtVerifier, startHeight int) (*Blockchain, error) {
	bc := &Blockchain{
		bcStore:        bcStore,
		accountStateDB: accountStateDB,
		engine:         engine,
		log:            log.GetLogger("blockchain"),
		debtVerifier:   verifier,
		lastBlockTime:  time.Now(),
	}

	var err error

	// recover from program crash
	bc.rp, err = loadRecoveryPoint(recoveryPointFile)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to load recovery point info from file %v", recoveryPointFile)
	}

	if err = bc.rp.recover(bcStore); err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to recover blockchain with RP %+v", *bc.rp)
	}

	// Get the genesis block from store
	genesisHash, err := bcStore.GetBlockHash(genesisBlockHeight)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get genesis block hash by height %v", genesisBlockHeight)
	}

	bc.genesisBlock, err = bcStore.GetBlock(genesisHash)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get genesis block by hash %v", genesisHash)
	}

	// Get the HEAD block from store
	var currentHeaderHash common.Hash
	if startHeight == -1 {
		currentHeaderHash, err = bcStore.GetHeadBlockHash()
		if err != nil {
			return nil, errors.NewStackedError(err, "failed to get HEAD block hash")
		}
	} else {
		// start from a specified height
		curHeight := uint64(startHeight)
		currentHeaderHash, err = bcStore.GetBlockHash(curHeight)
		if err != nil {
			return nil, errors.NewStackedError(err, "failed to get block hash at the specified height")
		}

		err = bcStore.PutHeadBlockHash(currentHeaderHash)
		if err != nil {
			return nil, errors.NewStackedError(err, "failed to update HEAD block hash")
		}
	}

	currentBlock, err := bcStore.GetBlock(currentHeaderHash)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get HEAD block by hash %v", currentHeaderHash)
	}
	bc.currentBlock.Store(currentBlock)

	// recover height-to-block mapping
	bc.recoverHeightIndices()

	td, err := bcStore.GetBlockTotalDifficulty(currentHeaderHash)
	if err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to get HEAD block TD by hash %v", currentHeaderHash)
	}

	blockIndex := NewBlockIndex(currentHeaderHash, currentBlock.Header.Height, td)
	bc.blockLeaves = NewBlockLeaves()
	bc.blockLeaves.Add(blockIndex)

	return bc, nil
}

// AccountDB returns the account state database in blockchain.
func (bc *Blockchain) AccountDB() database.Database {
	return bc.accountStateDB
}

// CurrentBlock returns the HEAD block of the blockchain.
func (bc *Blockchain) CurrentBlock() *types.Block {
	return bc.currentBlock.Load().(*types.Block)
}

// UpdateCurrentBlock updates the HEAD block of the blockchain.
func (bc *Blockchain) UpdateCurrentBlock(block *types.Block) {
	bc.currentBlock.Store(block)
}

func (bc *Blockchain) AddBlockLeaves(blockIndex *BlockIndex) {
	bc.blockLeaves.Add(blockIndex)
}

func (bc *Blockchain) RemoveBlockLeaves(hash common.Hash) {
	bc.blockLeaves.Remove(hash)
}

// CurrentHeader returns the HEAD block header of the blockchain.
func (bc *Blockchain) CurrentHeader() *types.BlockHeader {
	return bc.CurrentBlock().Header
}

// GetCurrentState returns the state DB of the current block.
func (bc *Blockchain) GetCurrentState() (*state.Statedb, error) {
	block := bc.CurrentBlock()
	return state.NewStatedb(block.Header.StateHash, bc.accountStateDB)
}

// GetHeaderByHeight retrieves a block header by height.
func (bc *Blockchain) GetHeaderByHeight(height uint64) *types.BlockHeader {
	hash, err := bc.bcStore.GetBlockHash(height)
	if err != nil {
		bc.log.Debug("get block header by height failed, err %s. height %d", err, height)
		return nil
	}

	return bc.GetHeaderByHash(hash)
}

// GetHeaderByHash retrieves a block header by hash.
func (bc *Blockchain) GetHeaderByHash(hash common.Hash) *types.BlockHeader {
	header, err := bc.bcStore.GetBlockHeader(hash)
	if err != nil {
		bc.log.Warn("get block header by hash failed, err %s, hash: %v", err, hash)
		return nil
	}

	return header
}

// GetBlockByHash retrieves a block by hash.
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

// GetStateByRootAndBlockHash will panic, since not supported
func (bc *Blockchain) GetStateByRootAndBlockHash(root, blockHash common.Hash) (*state.Statedb, error) {
	panic("unsupported")
}

// Genesis returns the genesis block of blockchain.
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
func (bc *Blockchain) WriteBlock(block *types.Block, txPool *Pool) error {
	startWriteBlockTime := time.Now()
	if err := bc.doWriteBlock(block, txPool); err != nil {
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

func (bc *Blockchain) doWriteBlock(block *types.Block, pool *Pool) error {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	auditor := log.NewAuditor(bc.log)
	auditor.AuditEnter("doWriteBlock")
	auditor.Audit("elapse since last block: %v", time.Since(bc.lastBlockTime))
	defer auditor.AuditLeave()

	// validate block
	if err := bc.validateBlock(block); err != nil {
		return errors.NewStackedError(err, "failed to validate block")
	}
	auditor.Audit("succeed to validate block %v", block.HeaderHash)

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
	auditor.Audit("succeed to apply %v txs and %v debts", len(block.Transactions), len(block.Debts))

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
	auditor.Audit("succeed to commit statedb changes to batch")

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
	for i, tx := range block.Transactions { // for 1st tx is reward tx, no need to check the duplicate
		if i == 0 {
			continue
		}
		if !pool.cachedTxs.has(tx.Hash) {
			bc.log.Debug("[CachedTxs] add tx %+v from synced block", tx.Hash)
			pool.cachedTxs.add(tx)
		}
	}

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
	isHead := true
	auditor.Audit("succeed to prepare block index")

	/////////////////////////////////////////////////////////////////
	// PAY ATTENTION TO THE ORDER OF WRITING DATA INTO DB.
	// OTHERWISE, THERE MAY BE INCONSISTENT DATA.
	// 1. Write account states
	// 2. Write receipts
	// 3. Write block
	/////////////////////////////////////////////////////////////////
	if err = batch.Commit(); err != nil {
		return errors.NewStackedError(err, "failed to batch commit statedb changes to database")
	}
	auditor.Audit("succeed to batch commit statedb chanages to database")

	if err = bc.rp.onPutBlockStart(block, bc.bcStore, isHead); err != nil {
		return errors.NewStackedErrorf(err, "failed to set recovery point before put block into store, isNewHead = %v", isHead)
	}

	if err = bc.bcStore.PutReceipts(block.HeaderHash, receipts); err != nil {
		return errors.NewStackedErrorf(err, "failed to save receipts into store, blockHash = %v, receipts count = %v", block.HeaderHash, len(receipts))
	}

	if err = bc.bcStore.PutBlock(block, currentTd, isHead); err != nil {
		return errors.NewStackedErrorf(err, "failed to save block into store, blockHash = %v, newTD = %v, isNewHead = %v", block.HeaderHash, currentTd, isHead)
	}
	auditor.Audit("succeed to save block into store, newHead = %v", isHead)

	bc.rp.onPutBlockEnd()

	// If the new block has larger TD, the canonical chain will be changed.
	// In this case, need to update the height-to-blockHash mapping for the new canonical chain.
	if isHead {
		largerHeight := block.Header.Height + 1
		if err = DeleteLargerHeightBlocks(bc.bcStore, largerHeight, bc.rp); err != nil {
			return errors.NewStackedErrorf(err, "failed to delete larger height blocks, height = %v", largerHeight)
		}
		auditor.Audit("succeed to delete larger height blocks, height = %v", largerHeight)

		previousHash := block.Header.PreviousBlockHash
		if err = OverwriteStaleBlocks(bc.bcStore, previousHash, bc.rp); err != nil {
			return errors.NewStackedErrorf(err, "failed to overwrite stale blocks, hash = %v", previousHash)
		}
		auditor.Audit("succeed to overwrite stale blocks, hash = %v", previousHash)
	}

	// update block header after meta info updated
	bc.blockLeaves.Add(blockIndex)
	bc.blockLeaves.Remove(block.Header.PreviousBlockHash)
	auditor.Audit("succeed to update block index, new = %v, old = %v", blockIndex.blockHash, block.Header.PreviousBlockHash)

	committed = true
	if isHead {
		//fmt.Printf("store currentBlock: %d", currentBlock.Header.Height)
		bc.currentBlock.Store(currentBlock)

		bc.blockLeaves.PurgeAsync(bc.bcStore, func(err error) {
			if err != nil {
				bc.log.Error(errors.NewStackedError(err, "failed to purge block").Error())
			}
		})

		event.ChainHeaderChangedEventMananger.Fire(block)
	}

	bc.lastBlockTime = time.Now()

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

	if len(block.Transactions) == 0 {
		return ErrBlockEmptyTxs
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

	if err := engine.VerifyHeader(chainReader, header); err != nil {
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
	auditor := log.NewAuditor(bc.log)

	statedb, err := state.NewStatedb(root, bc.accountStateDB)
	if err != nil {
		return nil, nil, errors.NewStackedErrorf(err, "failed to create statedb by root hash %v", root)
	}

	//validate debts
	// fix the issue caused by forking from collapse database
	if block.Height() > common.HeightRoof || block.Height() < common.HeightFloor {
		err = types.BatchValidateDebt(block.Debts, bc.debtVerifier)
		if err != nil {
			return nil, nil, errors.NewStackedError(err, "failed to batch validate debt")
		}
	}

	// update debts
	for _, d := range block.Debts {
		err = bc.ApplyDebtWithoutVerify(statedb, d, block.Header.Creator)
		if err != nil {
			return nil, nil, errors.NewStackedError(err, "failed to apply debt")
		}
	}
	auditor.Audit("succeed to validate %v debts", len(block.Debts))

	// apply txs
	receipts, err := bc.applyRewardAndRegularTxs(statedb, block.Transactions[0], block.Transactions[1:], block.Header)
	if err != nil {
		return nil, nil, errors.NewStackedErrorf(err, "failed to apply reward and regular txs")
	}
	auditor.Audit("succeed to update stateDB for %v txs", len(block.Transactions))

	return statedb, receipts, nil
}

func (bc *Blockchain) applyRewardAndRegularTxs(statedb *state.Statedb, rewardTx *types.Transaction, regularTxs []*types.Transaction, blockHeader *types.BlockHeader) ([]*types.Receipt, error) {
	auditor := log.NewAuditor(bc.log)

	receipts := make([]*types.Receipt, len(regularTxs)+1)

	// validate and apply reward txs
	if err := txs.ValidateRewardTx(rewardTx, blockHeader); err != nil {
		return nil, errors.NewStackedError(err, "failed to validate reward tx")
	}

	rewardReceipt, err := txs.ApplyRewardTx(rewardTx, statedb)
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to apply reward tx")
	}
	receipts[0] = rewardReceipt
	auditor.Audit("succeed to validate and apply reward tx")

	// batch validate signature to improve perf
	if err := types.BatchValidateTxs(regularTxs); err != nil {
		return nil, errors.NewStackedErrorf(err, "failed to batch validate %v txs", len(regularTxs))
	}
	auditor.Audit("succeed to batch validate (signature) %v txs", len(regularTxs))

	// process regular txs
	for i, tx := range regularTxs {
		txIdx := i + 1

		if err := tx.ValidateState(statedb, blockHeader.Height); err != nil {
			return nil, errors.NewStackedErrorf(err, "failed to validate tx[%v] against statedb", txIdx)
		}

		receipt, err := bc.ApplyTransaction(tx, txIdx, blockHeader.Creator, statedb, blockHeader)
		if err != nil {
			return nil, errors.NewStackedErrorf(err, "failed to apply tx[%v]", txIdx)
		}

		receipts[txIdx] = receipt
	}
	auditor.Audit("succeed to apply %v txs", len(regularTxs))

	return receipts, nil
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
	receipt, err := svm.Process(ctx, blockHeader.Height)
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to process tx via svm")
	}

	return receipt, nil
}

// ApplyDebtWithoutVerify applies a debt and update statedb.
func (bc *Blockchain) ApplyDebtWithoutVerify(statedb *state.Statedb, d *types.Debt, coinbase common.Address) error {
	debtIndex, _ := bc.bcStore.GetDebtIndex(d.Hash)
	if debtIndex != nil {
		return fmt.Errorf("debt already packed, debt hash %s", d.Hash.Hex())
	}

	if !statedb.Exist(d.Data.Account) {
		statedb.CreateAccount(d.Data.Account)
	}

	// @todo handle contract
	if d.Data.Amount == nil {
		return types.ErrAmountNil
	}

	if d.Data.Amount.Sign() < 0 {
		return types.ErrAmountNegative
	}

	if d.Fee() == nil {
		return types.ErrAmountNil
	}

	if d.Fee().Sign() < 0 {
		return types.ErrAmountNegative
	}

	statedb.AddBalance(d.Data.Account, d.Data.Amount)
	statedb.AddBalance(coinbase, d.Fee())

	return nil
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
	if err == nil {
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

// The following functions are only supported in lightclient
func (bc *Blockchain) PutCurrentHeader(header *types.BlockHeader) {
	panic("Not Supported")
}

func (bc *Blockchain) PutTd(td *big.Int) {
	panic("Not Supported")
}

func (bc *Blockchain) GetHeadRollbackEventManager() *event.EventManager {
	panic("Not Supported")
}

func (bc *Blockchain) recoverHeightIndices() {
	bc.log.Info("checking blockchain database...")
	curBlock := bc.CurrentBlock()
	curHeight := curBlock.Header.Height
	chainHeight := curHeight
	curHash := curBlock.Header.Hash()
	numGetBlockByHeight := 0
	numGetBlockByHash := 0
	numIrrecoverable := 0
	for curHeight > 0 {
		bc.log.Debug("checking blockchain database, height: %d", curHeight)
		if curBlock, err := bc.bcStore.GetBlockByHeight(curHeight); err != nil {
			bc.log.Error("height: %d, can't get block by height.", curHeight)
			if curBlock, err = bc.bcStore.GetBlock(curHash); err != nil {
				bc.log.Error("height: %d, can't get block by hash %v.", curHeight, curHash)
				curHash = common.EmptyHash
				numIrrecoverable++
			} else {
				// get block by hash successfully
				// recover the heightToBlock map
				bc.log.Info("height: %d, try to recover block by hash %v.", curHeight, curHash)
				if err := bc.bcStore.RecoverHeightToBlockMap(curBlock); err != nil {
					bc.log.Error("height: %d, can't recover block by hash %v.", curHeight, curHash)
				}
				curHash = curBlock.Header.PreviousBlockHash
				numGetBlockByHash++
			}
		} else {
			// get block by height successfully
			curHash = curBlock.Header.PreviousBlockHash
			numGetBlockByHeight++
		}
		curHeight--
	}
	bc.log.Info("Blockchain database checked, chainHeight: %d, numGetBlockByHeight: %d, numGetBlockByHash: %d, numIrrecoverable: %d", chainHeight, numGetBlockByHeight, numGetBlockByHash, numIrrecoverable)
}
