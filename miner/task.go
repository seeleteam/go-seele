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
	debts    []*types.Debt

	createdAt time.Time
	coinbase  common.Address
}

// applyTransactionsAndDebts TODO need to check more about the transactions, such as gas limit
func (task *Task) applyTransactionsAndDebts(seele SeeleBackend, statedb *state.Statedb, log *log.SeeleLog) error {
	// the reward tx will always be at the first of the block's transactions
	reward, err := task.handleMinerRewardTx(statedb)
	if err != nil {
		return err
	}

	// choose transactions from the given txs
	size := task.chooseDebts(seele, statedb, log)
	task.chooseTransactions(seele, statedb, log, size)

	log.Info("mining block height:%d, reward:%s, transaction number:%d", task.header.Height, reward, len(task.txs))

	root, err := statedb.Hash()
	if err != nil {
		return err
	}

	task.header.StateHash = root

	return nil
}

func (task *Task) chooseDebts(seele SeeleBackend, statedb *state.Statedb, log *log.SeeleLog) int {
	size := core.BlockByteLimit

	var debts []*types.Debt
	for size > 0 {
		debts, size = seele.DebtPool().Get(size)
		if len(debts) == 0 {
			return size
		}

		for _, d := range debts {
			// @TODO validate debt

			// @TODO add debt reward

			if !statedb.Exist(d.Data.Account) {
				statedb.CreateAccount(d.Data.Account)
			}

			statedb.AddBalance(d.Data.Account, d.Data.Amount)
		}
	}

	return size
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

func (task *Task) chooseTransactions(seele SeeleBackend, statedb *state.Statedb, log *log.SeeleLog, size int) {
	txIndex := 1 // the first tx is miner reward

	for size > 0 {
		txs, txsSize := seele.TxPool().GetProcessableTransactions(size)
		if len(txs) == 0 {
			break
		}

		for _, tx := range txs {
			if err := tx.Validate(statedb); err != nil {
				seele.TxPool().RemoveTransaction(tx.Hash)
				log.Error("failed to validate tx %s, for %s", tx.Hash.ToHex(), err)
				txsSize = txsSize - tx.Size()
				continue
			}

			receipt, err := seele.BlockChain().ApplyTransaction(tx, txIndex, task.coinbase, statedb, task.header)
			if err != nil {
				seele.TxPool().RemoveTransaction(tx.Hash)
				log.Error("failed to apply tx %s, %s", tx.Hash.ToHex(), err)
				txsSize = txsSize - tx.Size()
				continue
			}

			task.txs = append(task.txs, tx)
			task.receipts = append(task.receipts, receipt)
			txIndex++
		}

		size -= txsSize
	}
}

// generateBlock builds a block from task
func (task *Task) generateBlock() *types.Block {
	return types.NewBlock(task.header, task.txs, task.receipts, task.debts)
}

// Result is the result mined by engine. It contains the raw task and mined block.
type Result struct {
	task  *Task
	block *types.Block // mined block, with good nonce
}
