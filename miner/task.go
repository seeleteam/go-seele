/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math/big"
	"time"

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
func (task *Task) applyTransactions(seele SeeleBackend, statedb *state.Statedb,
	txs map[common.Address][]*types.Transaction, log *log.SeeleLog) error {
	// the reward tx will always be at the first of the block's transactions
	reward, err := task.handleMinerRewardTx(statedb)
	if err != nil {
		return err
	}

	// choose transactions from the given txs
	task.chooseTransactions(seele, statedb, txs, log)

	log.Info("mining block height:%d, reward:%s, transaction number:%d", task.header.Height, reward, len(task.txs))

	root, err := statedb.Commit(nil)
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

	stateObj := statedb.GetOrNewStateObject(task.coinbase)
	stateObj.AddAmount(reward)

	task.txs = append(task.txs, rewardTx)

	// add the receipt of the reward tx
	task.receipts = append(task.receipts, types.MakeRewardReceipt(rewardTx))

	return reward, nil
}

func (task *Task) chooseTransactions(seele SeeleBackend, statedb *state.Statedb, txs map[common.Address][]*types.Transaction, log *log.SeeleLog) {
	log.Debug("choose transaction from %d transactions in tx pool", len(txs))

	for i := 0; i < core.BlockTransactionNumberLimit-1; {
		tx := popBestFeeTx(txs)

		if tx == nil {
			break
		}

		seele.TxPool().UpdateTransactionStatus(tx.Hash, core.PROCESSING)

		err := tx.Validate(statedb)
		if err != nil {
			seele.TxPool().UpdateTransactionStatus(tx.Hash, core.ERROR)
			log.Error("validate tx %s failed, for %s", tx.Hash.ToHex(), err)
			continue
		}

		balance := statedb.GetBalance(tx.Data.From)
		nonce := statedb.GetNonce(tx.Data.From)
		receipt, err := seele.BlockChain().ApplyTransaction(tx, i+1, task.coinbase, statedb, task.header)
		log.Debug("miner apply account %s, balance transform %s -> %s, amount %s, nonce transaform %s -> %s",
			tx.Data.From.ToHex(), balance, statedb.GetBalance(tx.Data.From), tx.Data.Amount, nonce, statedb.GetNonce(tx.Data.From))

		if err != nil {
			seele.TxPool().UpdateTransactionStatus(tx.Hash, core.ERROR)
			log.Error("apply tx %s failed, %s", tx.Hash.ToHex(), err)
			continue
		}

		task.txs = append(task.txs, tx)
		task.receipts = append(task.receipts, receipt)

		i++
	}
}

// get best fee transaction and remove it in the map
// return best transaction if txs is empty, it will return nil
func popBestFeeTx(txs map[common.Address][]*types.Transaction) *types.Transaction {
	bestFee := big.NewInt(-1)
	var bestTx *types.Transaction
	for _, txSlice := range txs {
		if len(txSlice) > 0 {
			if txSlice[0].Data.Fee.Cmp(bestFee) > 0 {
				bestTx = txSlice[0]
				bestFee.Set(txSlice[0].Data.Fee)
			}
		}
	}

	if bestTx != nil {
		txSlice := txs[bestTx.Data.From]
		txs[bestTx.Data.From] = append(txSlice[:0], txSlice[1:]...)
	}

	return bestTx
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
