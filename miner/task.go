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
	coinbase  common.Address
}

// applyTransactions TODO need to check more about the transactions, such as gas limit
func (task *Task) applyTransactions(seele SeeleBackend, statedb *state.Statedb, blockHeight uint64,
	txs map[common.Address][]*types.Transaction, log *log.SeeleLog) error {
	// the reward tx will always be at the first of the block's transactions
	rewardValue := pow.GetReward(blockHeight)
	reward, err := types.NewTransaction(common.Address{}, task.coinbase, rewardValue, big.NewInt(0), 0)
	if err != nil {
		return err
	}
	reward.Signature = &crypto.Signature{}
	stateObj := statedb.GetOrNewStateObject(task.coinbase)
	stateObj.AddAmount(rewardValue)
	task.txs = append(task.txs, reward)

	// add the receipt of the reward tx
	task.receipts = append(task.receipts, types.MakeRewardReceipt(reward))

	task.chooseTransactions(seele, statedb, txs, log)

	log.Info("mining block height:%d, reward:%s, transaction number:%d", blockHeight, rewardValue, len(task.txs))

	root, err := statedb.Commit(nil)
	if err != nil {
		return err
	}

	task.header.StateHash = root

	return nil
}

func (task *Task) chooseTransactions(seele SeeleBackend, statedb *state.Statedb, txs map[common.Address][]*types.Transaction, log *log.SeeleLog) {
	for i := 0; i < core.BlockTransactionNumberLimit-1; {
		tx := popBestFeeTx(txs)
		if tx == nil {
			break
		}

		seele.TxPool().RemoveTransaction(tx.Hash)

		err := tx.Validate(statedb)
		if err != nil {
			log.Error("validating tx failed, for %s", err.Error())
			continue
		}

		receipt, err := seele.BlockChain().ApplyTransaction(tx, i+1, task.coinbase, statedb, task.header)
		if err != nil {
			log.Error("apply tx failed, %s", err.Error())
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
	bestFee := big.NewInt(0)
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
