/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

const transactionTimeoutDuration = 3 * time.Hour

// TransactionPool is a thread-safe container for transactions received from the network or submitted locally.
// A transaction will be removed from the pool once included in a blockchain or pending time too long (> transactionTimeoutDuration).
type TransactionPool struct {
	*Pool
}

// NewTransactionPool creates and returns a transaction pool.
func NewTransactionPool(config TransactionPoolConfig, chain blockchain) *TransactionPool {
	pool := NewPool(config.Capacity, chain, func(block *types.Block) []poolObject {
		var results []poolObject
		for _, obj := range block.GetExcludeRewardTransactions() {
			results = append(results, obj)
		}

		return results
	}, func(chain blockchain, state *state.Statedb, log *log.SeeleLog, item *poolItem) bool {
		nowTimestamp := time.Now()
		txIndex, _ := chain.GetStore().GetTxIndex(item.GetHash())
		nonce := state.GetNonce(item.Account())
		duration := nowTimestamp.Sub(item.timestamp)

		// Transactions have been processed or are too old need to delete
		if txIndex != nil || item.Nonce() < nonce || duration > transactionTimeoutDuration {
			if txIndex == nil {
				if item.Nonce() < nonce {
					log.Debug("remove tx %s because nonce too low, account %s, tx nonce %d, target nonce %d", item.GetHash().ToHex(),
						item.Account().ToHex(), item.Nonce(), nonce)
				} else if duration > transactionTimeoutDuration {
					log.Debug("remove tx %s because not packed for more than three hours", item.GetHash().ToHex())
				}
			}

			return true
		}

		return false
	})

	return &TransactionPool{pool}
}

func (pool *TransactionPool) AddTransaction(tx *types.Transaction) error {
	if tx == nil {
		return nil
	}

	if pool.Has(tx.GetHash()) {
		return errObjectHashExists
	}

	// validate tx against the latest statedb
	statedb, err := pool.chain.GetCurrentState()
	if err != nil {
		return errors.NewStackedError(err, "failed to get current statedb")
	}

	if err := tx.Validate(statedb); err != nil {
		return errors.NewStackedError(err, "failed to validate tx")
	}

	return pool.AddObject(tx)
}

func (pool *TransactionPool) GetTransaction(txHash common.Hash) *types.Transaction {
	obj := pool.GetObject(txHash)
	return obj.(*types.Transaction)
}

func (pool *TransactionPool) RemoveTransaction(txHash common.Hash) {
	pool.RemoveOject(txHash)
}

func (pool *TransactionPool) GetProcessableTransactions(size int) ([]*types.Transaction, int) {
	objects, size := pool.GetProcessableObjects(size)
	return poolObjectToTxs(objects), size
}

func (pool *TransactionPool) GetPendingTxCount() int {
	return pool.GetPendingObjectCount()
}

func (pool *TransactionPool) GetTxCount() int {
	return pool.GetObjectCount()
}

func (pool *TransactionPool) GetTransactions(processing, pending bool) []*types.Transaction {
	objects := pool.GetObjects(processing, pending)
	return poolObjectToTxs(objects)
}

func poolObjectToTxs(objects []poolObject) []*types.Transaction {
	var txs []*types.Transaction
	for _, obj := range objects {
		txs = append(txs, obj.(*types.Transaction))
	}

	return txs
}
