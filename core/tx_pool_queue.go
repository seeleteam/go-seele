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

type heapedTxList struct {
	common.BaseHeapItem
	*txCollection
}

type heapedTxListPair struct {
	best  *heapedTxList
	worst *heapedTxList
}

// pendingQueue represents the heaped transactions that grouped by account.
type pendingQueue struct {
	txs       map[common.Address]*heapedTxListPair
	bestHeap  *common.Heap
	worstHeap *common.Heap
}

func newPendingQueue() *pendingQueue {
	return &pendingQueue{
		txs: make(map[common.Address]*heapedTxListPair),
		bestHeap: common.NewHeap(func(i, j common.HeapItem) bool {
			iCollection := i.(*heapedTxList).txCollection
			jCollection := j.(*heapedTxList).txCollection
			return iCollection.cmp(jCollection) > 0
		}),
		worstHeap: common.NewHeap(func(i, j common.HeapItem) bool {
			iCollection := i.(*heapedTxList).txCollection
			jCollection := j.(*heapedTxList).txCollection
			return iCollection.cmp(jCollection) <= 0
		}),
	}
}

func (q *pendingQueue) add(tx *pooledTx) {
	if pair := q.txs[tx.Data.From]; pair != nil {
		pair.best.add(tx)

		heap.Fix(q.bestHeap, pair.best.GetHeapIndex())
		heap.Fix(q.worstHeap, pair.worst.GetHeapIndex())
	} else {
		collection := newTxCollection()
		collection.add(tx)

		pair := &heapedTxListPair{
			best:  &heapedTxList{txCollection: collection},
			worst: &heapedTxList{txCollection: collection},
		}

		q.txs[tx.Data.From] = pair
		heap.Push(q.bestHeap, pair.best)
		heap.Push(q.worstHeap, pair.worst)
	}
}

func (q *pendingQueue) get(addr common.Address, nonce uint64) *pooledTx {
	pair := q.txs[addr]
	if pair == nil {
		return nil
	}

	return pair.best.get(nonce)
}

func (q *pendingQueue) remove(addr common.Address, nonce uint64) {
	pair := q.txs[addr]
	if pair == nil {
		return
	}

	if !pair.best.remove(nonce) {
		return
	}

	if pair.best.len() == 0 {
		delete(q.txs, addr)
		heap.Remove(q.bestHeap, pair.best.GetHeapIndex())
		heap.Remove(q.worstHeap, pair.worst.GetHeapIndex())
	} else {
		heap.Fix(q.bestHeap, pair.best.GetHeapIndex())
		heap.Fix(q.worstHeap, pair.worst.GetHeapIndex())
	}
}

func (q *pendingQueue) count() int {
	sum := 0

	for _, pair := range q.txs {
		sum += pair.best.len()
	}

	return sum
}

func (q *pendingQueue) empty() bool {
	return q.bestHeap.Len() == 0
}

func (q *pendingQueue) peek() *txCollection {
	if item := q.bestHeap.Peek(); item != nil {
		return item.(*heapedTxList).txCollection
	}

	return nil
}

func (q *pendingQueue) popN(n int) []*types.Transaction {
	var txs []*types.Transaction

	for i := 0; i < n && q.bestHeap.Len() > 0; i++ {
		txs = append(txs, q.pop())
	}

	return txs
}

func (q *pendingQueue) pop() *types.Transaction {
	tx := q.bestHeap.Peek().(*heapedTxList).pop().Transaction
	pair := q.txs[tx.Data.From]

	if pair.best.len() == 0 {
		delete(q.txs, tx.Data.From)
		heap.Remove(q.bestHeap, pair.best.GetHeapIndex())
		heap.Remove(q.worstHeap, pair.worst.GetHeapIndex())
	} else {
		heap.Fix(q.bestHeap, pair.best.GetHeapIndex())
		heap.Fix(q.worstHeap, pair.worst.GetHeapIndex())
	}

	return tx
}

func (q *pendingQueue) peekWorst() *txCollection {
	if item := q.worstHeap.Peek(); item != nil {
		return item.(*heapedTxList).txCollection
	}

	return nil
}

// discard removes and returns the txs of worst account.
func (q *pendingQueue) discard() *txCollection {
	if q.worstHeap.Len() == 0 {
		return nil
	}

	// pop the worst txs
	item := heap.Pop(q.worstHeap).(*heapedTxList).txCollection

	// peek will never return nil, since empty txCollection
	// will be removed when remove() or pop() invoked.
	account := item.peek().Data.From

	// remove the txs from best heap and txs of discarded account.
	heap.Remove(q.bestHeap, q.txs[account].best.GetHeapIndex())
	delete(q.txs, account)

	return item
}

func (q *pendingQueue) list() []*types.Transaction {
	var result []*types.Transaction

	for _, pair := range q.txs {
		result = append(result, pair.best.list()...)
	}

	return result
}
