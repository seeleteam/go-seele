/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"container/heap"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

// pendingQueue represents the fee sorted transactions that grouped by account.
type pendingQueue struct {
	txs     map[common.Address]*txCollection
	feeHeap *common.Heap
}

func newPendingQueue() *pendingQueue {
	return &pendingQueue{
		txs: make(map[common.Address]*txCollection),
		feeHeap: common.NewHeap(func(i, j common.HeapItem) bool {
			iCollection, jCollection := i.(*txCollection), j.(*txCollection)
			return iCollection.cmp(jCollection) > 0
		}),
	}
}

func (q *pendingQueue) add(tx *pooledTx) {
	if collection := q.txs[tx.Data.From]; collection != nil {
		if updated := !collection.add(tx); updated {
			heap.Fix(q.feeHeap, collection.GetHeapIndex())
		}
	} else {
		collection := newTxCollection()
		collection.add(tx)

		q.txs[tx.Data.From] = collection
		heap.Push(q.feeHeap, collection)
	}
}

func (q *pendingQueue) get(addr common.Address, nonce uint64) *pooledTx {
	collection := q.txs[addr]
	if collection == nil {
		return nil
	}

	return collection.get(nonce)
}

func (q *pendingQueue) remove(addr common.Address, nonce uint64) {
	collection := q.txs[addr]
	if collection == nil {
		return
	}

	if !collection.remove(nonce) {
		return
	}

	if collection.len() == 0 {
		delete(q.txs, addr)
		heap.Remove(q.feeHeap, collection.GetHeapIndex())
	} else {
		heap.Fix(q.feeHeap, collection.GetHeapIndex())
	}
}

func (q *pendingQueue) count() int {
	sum := 0

	for _, collection := range q.txs {
		sum += collection.len()
	}

	return sum
}

func (q *pendingQueue) peek() *txCollection {
	if item := q.feeHeap.Peek(); item != nil {
		return item.(*txCollection)
	}

	return nil
}

func (q *pendingQueue) popN(n int) []*types.Transaction {
	var txs []*types.Transaction

	for i := 0; i < n && q.feeHeap.Len() > 0; i++ {
		txs = append(txs, q.pop())
	}

	return txs
}

func (q *pendingQueue) pop() *types.Transaction {
	if q.feeHeap.Len() == 0 {
		return nil
	}

	collection := q.peek()
	tx := collection.pop().Transaction

	if collection.len() == 0 {
		delete(q.txs, tx.Data.From)
		heap.Remove(q.feeHeap, collection.GetHeapIndex())
	} else {
		heap.Fix(q.feeHeap, collection.GetHeapIndex())
	}

	return tx
}

func (q *pendingQueue) list() []*types.Transaction {
	var result []*types.Transaction

	for _, collection := range q.txs {
		result = append(result, collection.list()...)
	}

	return result
}
