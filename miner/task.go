/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"errors"
	"math/big"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner/pow"
	"github.com/seeleteam/go-seele/seele"
)

var ErrNotEnoughTransactions = errors.New("not enough transactions")

// Task is a mining work for engine, it contains block header, transactions, and transaction receipts.
type Task struct {
	header *types.BlockHeader
	txs    []*types.Transaction

	createdAt time.Time
}

// applyTransactions TODO need check more about the transactions, such as gas limit
func (task *Task) applyTransactions(seele *seele.SeeleService, statedb *state.Statedb, txs []*types.Transaction, log *log.SeeleLog) error {
	// the reward tx will always be at the first of the block's transactions
	rewardValue := big.NewInt(pow.MinerRewardAmount)
	reward := types.NewTransaction(common.Address{}, seele.Coinbase, rewardValue, 0)
	reward.Signature = &crypto.Signature{}
	stateObj := statedb.GetOrNewStateObject(seele.Coinbase)
	stateObj.AddAmount(rewardValue)
	task.txs = append(task.txs, reward)

	for _, tx := range txs {
		seele.TxPool().RemoveTransaction(tx.Hash)

		err := tx.Validate(statedb)
		if err != nil {
			log.Error("exec tx failed, cause for %s", err)
			continue
		}

		fromStateObj := statedb.GetOrNewStateObject(tx.Data.From)
		fromStateObj.SubAmount(tx.Data.Amount)
		fromStateObj.SetNonce(tx.Data.AccountNonce + 1)

		toStateObj := statedb.GetOrNewStateObject(*tx.Data.To)
		toStateObj.AddAmount(tx.Data.Amount)

		task.txs = append(task.txs, tx)
	}

	if len(task.txs) == 0 {
		return ErrNotEnoughTransactions
	}

	log.Info("miner transaction number %d", len(task.txs))

	root, err := statedb.Commit(nil)
	if err != nil {
		return err
	}
	task.header.StateHash = root

	return nil
}

func (task *Task) generateBlock() *types.Block {
	return types.NewBlock(task.header, task.txs)
}

// Result if struct of mined result by engine, it contains the raw task and mined block.
type Result struct {
	task  *Task
	block *types.Block // mined block, with good nonce
}
