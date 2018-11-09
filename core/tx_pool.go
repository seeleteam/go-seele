/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/types"
)

// Pool is a thread-safe container for transactions received
// from the network or submitted locally. A transaction will be removed from
// the pool once included in a blockchain or pending time too long (> overTimeInterval).
type TransactionPool struct {
	*Pool
}

// NewPool creates and returns a transaction pool.
func NewTransactionPool(config TransactionPoolConfig, chain blockchain) *TransactionPool {
	pool := NewPool(config.Capacity, chain)

	return &TransactionPool{pool}
}

func (pool *TransactionPool) AddTransaction(tx *types.Transaction) error {
	if tx == nil {
		return nil
	}

	if pool.Has(tx.GetHash()) {
		return errTxHashExists
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
