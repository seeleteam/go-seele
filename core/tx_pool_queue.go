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

type txHeapByFee []*txCollection

func (h txHeapByFee) Len() int {
	return len(h)
}

func (h txHeapByFee) Less(i, j int) bool {
	iTx := h[i].peek()
	jTx := h[j].peek()

	r := iTx.Data.Fee.Cmp(jTx.Data.Fee)
	switch r {
	case -1:
		return false
	case 1:
		return true
	default:
		return iTx.timestamp.Before(jTx.timestamp)
	}
}

func (h txHeapByFee) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heapIndex = i
	h[j].heapIndex = j
}

func (h *txHeapByFee) Push(x interface{}) {
	q := x.(*txCollection)
	q.heapIndex = h.Len()
	*h = append(*h, q)
}

func (h *txHeapByFee) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// pendingQueue represents the fee sorted transactions that grouped by account.
type pendingQueue struct {
	txs     map[common.Address]*txCollection
	feeHeap txHeapByFee
}

func newPendingQueue() *pendingQueue {
	return &pendingQueue{
		txs: make(map[common.Address]*txCollection),
	}
}

func (q *pendingQueue) add(tx *pooledTx) {
	if collection := q.txs[tx.Data.From]; collection != nil {
		if updated := !collection.add(tx); updated {
			heap.Fix(&q.feeHeap, collection.heapIndex)
		}
	} else {
		collection := newTxCollection()
		collection.add(tx)

		q.txs[tx.Data.From] = collection
		heap.Push(&q.feeHeap, collection)
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
		heap.Remove(&q.feeHeap, collection.heapIndex)
	} else {
		heap.Fix(&q.feeHeap, collection.heapIndex)
	}
}

func (q *pendingQueue) count() int {
	sum := 0

	for _, collection := range q.feeHeap {
		sum += collection.len()
	}

	return sum
}

func (q *pendingQueue) peek() *txCollection {
	return q.feeHeap[0]
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
		heap.Remove(&q.feeHeap, collection.heapIndex)
	} else {
		heap.Fix(&q.feeHeap, collection.heapIndex)
	}

	return tx
}

func (q *pendingQueue) list() []*types.Transaction {
	var result []*types.Transaction

	for _, collection := range q.feeHeap {
		result = append(result, collection.list()...)
	}

	return result
}
