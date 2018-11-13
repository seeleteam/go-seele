/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"container/heap"
	"math/big"

	"github.com/seeleteam/go-seele/common"
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

func (q *pendingQueue) add(tx *poolItem) {
	if pair := q.txs[tx.FromAccount()]; pair != nil {
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

		q.txs[tx.FromAccount()] = pair
		heap.Push(q.bestHeap, pair.best)
		heap.Push(q.worstHeap, pair.worst)
	}
}

func (q *pendingQueue) get(addr common.Address, nonce uint64) *poolItem {
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

func (q *pendingQueue) popN(n int) []poolObject {
	var txs []poolObject

	for i := 0; i < n && q.bestHeap.Len() > 0; i++ {
		txs = append(txs, q.pop())
	}

	return txs
}

func (q *pendingQueue) pop() poolObject {
	tx := q.bestHeap.Peek().(*heapedTxList).pop().poolObject
	pair := q.txs[tx.FromAccount()]

	if pair.best.len() == 0 {
		delete(q.txs, tx.FromAccount())
		heap.Remove(q.bestHeap, pair.best.GetHeapIndex())
		heap.Remove(q.worstHeap, pair.worst.GetHeapIndex())
	} else {
		heap.Fix(q.bestHeap, pair.best.GetHeapIndex())
		heap.Fix(q.worstHeap, pair.worst.GetHeapIndex())
	}

	return tx
}

// discard removes and returns the txs of worst account that has
// lower price than the specified price. Return nil if no lower
// price txs found.
func (q *pendingQueue) discard(price *big.Int) *txCollection {
	if q.empty() {
		return nil
	}

	worstCollection := q.worstHeap.Peek().(*heapedTxList).txCollection
	worstTx := worstCollection.peek()
	if worstTx == nil || price.Cmp(worstTx.Price()) <= 0 {
		return nil
	}

	heap.Pop(q.worstHeap)
	account := worstTx.FromAccount()
	heap.Remove(q.bestHeap, q.txs[account].best.GetHeapIndex())
	delete(q.txs, account)

	return worstCollection
}

func (q *pendingQueue) list() []poolObject {
	var result []poolObject

	for _, pair := range q.txs {
		result = append(result, pair.best.list()...)
	}

	return result
}
