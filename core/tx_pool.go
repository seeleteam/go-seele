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
	"github.com/seeleteam/go-seele/event"
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
	log := log.GetLogger("txpool")
	getObjectFromBlock := func(block *types.Block) []poolObject {
		return txsToObjects(block.GetExcludeRewardTransactions())
	}

	canRemove := func(chain blockchain, state *state.Statedb, item *poolItem) bool {
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
	}

	objectValidation := func(state *state.Statedb, obj poolObject) error {
		tx := obj.(*types.Transaction)
		if err := tx.Validate(state); err != nil {
			return errors.NewStackedError(err, "failed to validate tx")
		}

		return nil
	}

	afterAdd := func(obj poolObject) {
		log.Debug("receive transaction and add it. transaction hash: %v, time: %d", obj.GetHash(), time.Now().UnixNano())

		// fire event
		event.TransactionInsertedEventManager.Fire(obj.(*types.Transaction))
	}

	pool := NewPool(config.Capacity, chain, getObjectFromBlock, canRemove, log, objectValidation, afterAdd)

	return &TransactionPool{pool}
}

func (pool *TransactionPool) AddTransaction(tx *types.Transaction) error {
	if tx == nil {
		return nil
	}

	return pool.addObject(tx)
}

func (pool *TransactionPool) GetTransaction(txHash common.Hash) *types.Transaction {
	obj := pool.GetObject(txHash)
	v, ok := obj.(*types.Transaction)
	if ok {
		return v
	}

	return nil
}

func (pool *TransactionPool) RemoveTransaction(txHash common.Hash) {
	pool.removeOject(txHash)
}

func (pool *TransactionPool) GetProcessableTransactions(size int) ([]*types.Transaction, int) {
	objects, size := pool.getProcessableObjects(size)
	return poolObjectToTxs(objects), size
}

func (pool *TransactionPool) GetPendingTxCount() int {
	return pool.getPendingObjectCount()
}

func (pool *TransactionPool) GetTxCount() int {
	return pool.getObjectCount()
}

func (pool *TransactionPool) GetTransactions(processing, pending bool) []*types.Transaction {
	objects := pool.getObjects(processing, pending)
	return poolObjectToTxs(objects)
}

func poolObjectToTxs(objects []poolObject) []*types.Transaction {
	txs := make([]*types.Transaction, len(objects))
	for index, obj := range objects {
		txs[index] = obj.(*types.Transaction)
	}

	return txs
}

func txsToObjects(txs []*types.Transaction) []poolObject {
	objects := make([]poolObject, len(txs))
	for index, tx := range txs {
		objects[index] = tx
	}

	return objects
}
