/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner/pow"
)

// Task is a mining work for engine, containing block header, transactions, and transaction receipts.
type Task struct {
	header   *types.BlockHeader
	txs      []*types.Transaction
	receipts []*types.Receipt

	createdAt time.Time
}

// applyTransactions TODO need to check more about the transactions, such as gas limit
func (task *Task) applyTransactions(seele SeeleBackend, statedb *state.Statedb, blockHeight uint64,
	txs []*types.Transaction, log *log.SeeleLog) error {
	// the reward tx will always be at the first of the block's transactions
	rewardValue := pow.GetReward(blockHeight)
	reward := types.NewTransaction(common.Address{}, seele.GetCoinbase(), rewardValue, 0)
	reward.Signature = &crypto.Signature{}
	stateObj := statedb.GetOrNewStateObject(seele.GetCoinbase())
	stateObj.AddAmount(rewardValue)
	task.txs = append(task.txs, reward)

	for i, tx := range txs {
		seele.TxPool().RemoveTransaction(tx.Hash)

		err := tx.Validate(statedb)
		if err != nil {
			log.Error("validating tx failed, for %s", err.Error())
			continue
		}

		receipt, err := seele.BlockChain().ApplyTransaction(tx, i, seele.GetCoinbase(), statedb, task.header)
		if err != nil {
			log.Error("apply tx failed, %s", err.Error())
			continue
		}

		task.txs = append(task.txs, tx)
		task.receipts = append(task.receipts, receipt)

		if i == core.BlockTransactionNumberLimit {
			break
		}
	}

	log.Info("mining block height:%d, reward:%s, transaction number:%d", blockHeight, rewardValue, len(task.txs))

	root, err := statedb.Commit(nil)
	if err != nil {
		return err
	}

	task.header.StateHash = root

	return nil
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
