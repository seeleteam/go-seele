/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"container/heap"
	"sort"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

// txCollection represents the nonce sorted transactions of an account.
type txCollection struct {
	common.BaseHeapItem
	txs       map[uint64]*pooledTx
	nonceHeap *common.Heap
}

func newTxCollection() *txCollection {
	return &txCollection{
		txs: make(map[uint64]*pooledTx),
		nonceHeap: common.NewHeap(func(i, j common.HeapItem) bool {
			iNonce := i.(*pooledTx).Data.AccountNonce
			jNonce := j.(*pooledTx).Data.AccountNonce
			return iNonce < jNonce
		}),
	}
}

func (collection *txCollection) add(tx *pooledTx) bool {
	if existTx := collection.txs[tx.Data.AccountNonce]; existTx != nil {
		existTx.Transaction = tx.Transaction
		existTx.timestamp = tx.timestamp
		return false
	}

	heap.Push(collection.nonceHeap, tx)
	collection.txs[tx.Data.AccountNonce] = tx

	return true
}

func (collection *txCollection) get(nonce uint64) *pooledTx {
	return collection.txs[nonce]
}

func (collection *txCollection) remove(nonce uint64) bool {
	if tx := collection.txs[nonce]; tx != nil {
		heap.Remove(collection.nonceHeap, tx.GetHeapIndex())
		delete(collection.txs, nonce)
		return true
	}

	return false
}

func (collection *txCollection) len() int {
	return collection.nonceHeap.Len()
}

func (collection *txCollection) peek() *pooledTx {
	if item := collection.nonceHeap.Peek(); item != nil {
		return item.(*pooledTx)
	}

	return nil
}

func (collection *txCollection) pop() *pooledTx {
	tx := heap.Pop(collection.nonceHeap).(*pooledTx)
	delete(collection.txs, tx.Data.AccountNonce)
	return tx
}

func (collection *txCollection) list() []*types.Transaction {
	result := make([]*types.Transaction, len(collection.txs))
	i := 0

	for _, tx := range collection.txs {
		result[i] = tx.Transaction
		i++
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Data.AccountNonce < result[j].Data.AccountNonce
	})

	return result
}

// cmp compares to the specified tx collection based on price and timestamp.
//   For higher price, return 1.
//   For lower price, return -1.
//   Otherwise:
//     For earier timestamp, return 1.
//     For later timestamp, return -1.
//     Otherwise, return 0.
func (collection *txCollection) cmp(other *txCollection) int {
	iTx, jTx := collection.peek(), other.peek()

	if r := iTx.Data.GasPrice.Cmp(jTx.Data.GasPrice); r != 0 {
		return r
	}

	if iTx.timestamp.Before(jTx.timestamp) {
		return 1
	}

	if iTx.timestamp.After(jTx.timestamp) {
		return -1
	}

	return 0
}
