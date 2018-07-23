/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"container/heap"
	"sort"

	"github.com/seeleteam/go-seele/core/types"
)

type heapItem struct {
	heapIndex int // used to randomly remove an item from heap.
}

type txHeapByNonce []*pooledTx

func (h txHeapByNonce) Len() int {
	return len(h)
}

func (h txHeapByNonce) Less(i, j int) bool {
	return h[i].Data.AccountNonce < h[j].Data.AccountNonce
}

func (h txHeapByNonce) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heapIndex = i
	h[j].heapIndex = j
}

func (h *txHeapByNonce) Push(x interface{}) {
	tx := x.(*pooledTx)
	tx.heapIndex = h.Len()
	*h = append(*h, tx)
}

func (h *txHeapByNonce) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// txCollection represents the nonce sorted transactions of an account.
type txCollection struct {
	heapItem
	txs       map[uint64]*pooledTx
	nonceHeap txHeapByNonce
}

func newTxCollection() *txCollection {
	return &txCollection{
		txs: make(map[uint64]*pooledTx),
	}
}

func (collection *txCollection) add(tx *pooledTx) bool {
	if existTx := collection.txs[tx.Data.AccountNonce]; existTx != nil {
		existTx.Transaction = tx.Transaction
		existTx.timestamp = tx.timestamp
		return false
	}

	heap.Push(&collection.nonceHeap, tx)
	collection.txs[tx.Data.AccountNonce] = tx

	return true
}

func (collection *txCollection) get(nonce uint64) *pooledTx {
	return collection.txs[nonce]
}

func (collection *txCollection) remove(nonce uint64) bool {
	if tx := collection.txs[nonce]; tx != nil {
		heap.Remove(&collection.nonceHeap, tx.heapIndex)
		delete(collection.txs, nonce)
		return true
	}

	return false
}

func (collection *txCollection) len() int {
	return collection.nonceHeap.Len()
}

func (collection *txCollection) peek() *pooledTx {
	return collection.nonceHeap[0]
}

func (collection *txCollection) pop() *pooledTx {
	tx := heap.Pop(&collection.nonceHeap).(*pooledTx)
	delete(collection.txs, tx.Data.AccountNonce)
	return tx
}

func (collection *txCollection) list() []*types.Transaction {
	result := make([]*types.Transaction, collection.len())
	for i, tx := range collection.nonceHeap {
		result[i] = tx.Transaction
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Data.AccountNonce < result[j].Data.AccountNonce
	})

	return result
}
