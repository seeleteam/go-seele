/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math/big"
	"time"
	"unsafe"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner/pow"
)

// Task is a mining work for engine, containing block header, transactions, and transaction receipts.
type Task struct {
	header   *types.BlockHeader
	txs      []*types.Transaction
	receipts []*types.Receipt

	createdAt time.Time
	coinbase  common.Address
}

// applyTransactions TODO need to check more about the transactions, such as gas limit
func (task *Task) applyTransactions(seele SeeleBackend, statedb *state.Statedb, log *log.SeeleLog) error {
	// the reward tx will always be at the first of the block's transactions
	reward, err := task.handleMinerRewardTx(statedb)
	if err != nil {
		return err
	}

	// choose transactions from the given txs
	task.chooseTransactions(seele, statedb, log)

	log.Info("mining block height:%d, reward:%s, transaction number:%d", task.header.Height, reward, len(task.txs))

	root, err := statedb.Hash()
	if err != nil {
		return err
	}

	task.header.StateHash = root

	return nil
}

// handleMinerRewardTx handles the miner reward transaction.
func (task *Task) handleMinerRewardTx(statedb *state.Statedb) (*big.Int, error) {
	reward := pow.GetReward(task.header.Height)
	rewardTx, err := types.NewRewardTransaction(task.coinbase, reward, task.header.CreateTimestamp.Uint64())
	if err != nil {
		return nil, err
	}

	rewardTxReceipt, err := core.ApplyRewardTx(rewardTx, statedb)
	if err != nil {
		return nil, err
	}

	task.txs = append(task.txs, rewardTx)

	// add the receipt of the reward tx
	task.receipts = append(task.receipts, rewardTxReceipt)

	return reward, nil
}

// getDataSize gets the transactions size
func getDataSize(data *types.Transaction) uint64 {
	size := uint64(0)
	var boolvalue bool
	boolSize := uint64(unsafe.Sizeof(boolvalue))
	hashSize := uint64(len(data.Hash))
	signSize := uint64(len(data.Signature.Sig))
	fromSize := uint64(len(data.Data.From))
	toSize := uint64(len(data.Data.To))
	amountSize := uint64(len(data.Data.Amount.Bits())) + boolSize
	nonceSize := uint64(unsafe.Sizeof(data.Data.AccountNonce))
	feeSize := uint64(len(data.Data.Fee.Bits())) + boolSize
	timestampSize := uint64(unsafe.Sizeof(data.Data.Timestamp))
	payloadSize := uint64(len(data.Data.Payload))

	size = size + hashSize + signSize + fromSize + toSize + amountSize + nonceSize + feeSize + timestampSize + payloadSize

	return size
}

// addTransanction add one to task
func addTransanction(task *Task, seele SeeleBackend, statedb *state.Statedb, log *log.SeeleLog, count int, tx *types.Transaction) bool {
	if err := tx.Validate(statedb); err != nil {
		seele.TxPool().RemoveTransaction(tx.Hash)
		log.Error("failed to validate tx %s, for %s", tx.Hash.ToHex(), err)
		return false
	}

	receipt, err := seele.BlockChain().ApplyTransaction(tx, count+1, task.coinbase, statedb, task.header)
	if err != nil {
		seele.TxPool().RemoveTransaction(tx.Hash)
		log.Error("failed to apply tx %s, %s", tx.Hash.ToHex(), err)
		return false
	}

	task.txs = append(task.txs, tx)
	task.receipts = append(task.receipts, receipt)
	return true
}

func (task *Task) chooseTransactions(seele SeeleBackend, statedb *state.Statedb, log *log.SeeleLog) {
	size := uint64(0)
	count := 0

	for {
		txs := seele.TxPool().GetProcessableTransactions(1)
		if len(txs) == 0 {
			break
		}

		tx := txs[0]
		txSize := getDataSize(tx)
		tmpSize := txSize + size
		if tmpSize > core.TransactionSizeLimit {
			addTransanction(task, seele, statedb, log, count, tx)
			break
		}

		if addTransanction(task, seele, statedb, log, count, tx) {
			count++
			size = tmpSize
		}
	}
}

// generateBlock builds a block from task
func (task *Task) generateBlock() *types.Block {
	return types.NewBlock(task.header, task.txs, task.receipts)
}

// Result is the result mined by engine. It contains the raw task and mined block.
type Result struct {
	task  *Task
	block *types.Block // mined block, with good nonce
}
