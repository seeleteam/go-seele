/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/seele"

	"errors"
	"time"
)

var ErrNotEnoughTransactions = errors.New("not enough transactions")

// Task is a mining work for engine, it contains block header, transactions, and transaction receipts.
type Task struct {
	header *types.BlockHeader
	txs    []*types.Transaction

	createdAt time.Time
}

// applyTransactions TODO need check more about the transactions, such as gas limit
func (task *Task) applyTransactions(seele *seele.SeeleService, coinbase common.Address, txs []*types.Transaction, log *log.SeeleLog) error {
	for _, tx := range txs {
		// execute tx
		err := seele.ApplyTransaction(coinbase, tx)
		if err != nil {
			log.Error("exec tx failed, cause for %s", err)
			continue
		}

		task.txs = append(task.txs, tx)
	}

	if len(task.txs) == 0 {
		return ErrNotEnoughTransactions
	}

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
