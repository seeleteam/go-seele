/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"container/heap"
)

// HeapItem is implemented by any type that support heap manipulations.
type HeapItem interface {
	getHeapIndex() int // to support delete item from heap by index.
	setHeapIndex(int)  // to record the item index in heap.
}

// BaseHeapItem is base struct of any heap item.
type BaseHeapItem struct {
	heapIndex int
}

func (item *BaseHeapItem) getHeapIndex() int {
	return item.heapIndex
}

func (item *BaseHeapItem) setHeapIndex(index int) {
	item.heapIndex = index
}

// Heap is a common used heap for generic type purpose.
type Heap struct {
	data     []HeapItem
	lessFunc func(HeapItem, HeapItem) bool
}

// NewHeap creates a new heap with specified Less func.
func NewHeap(lessFunc func(HeapItem, HeapItem) bool) *Heap {
	return &Heap{
		lessFunc: lessFunc,
	}
}

// Len implements the heap.Interface
func (h *Heap) Len() int {
	return len(h.data)
}

// Less implements the heap.Interface
func (h *Heap) Less(i, j int) bool {
	return h.lessFunc(h.data[i], h.data[j])
}

// Swap implements the heap.Interface
func (h *Heap) Swap(i, j int) {
	h.data[i], h.data[j] = h.data[j], h.data[i]
	h.data[i].setHeapIndex(i)
	h.data[j].setHeapIndex(j)
}

// Push implements the heap.Interface
func (h *Heap) Push(x interface{}) {
	item := x.(HeapItem)
	item.setHeapIndex(h.Len())
	h.data = append(h.data, item)
}

// Pop implements the heap.Interface
func (h *Heap) Pop() interface{} {
	n := h.Len()
	x := h.data[n-1]
	h.data = h.data[0 : n-1]
	return x
}

// Peek returns the top value in heap if any. Otherwise, return nil.
func (h *Heap) Peek() interface{} {
	if h.Len() == 0 {
		return nil
	}

	return h.data[0]
}

// Remove removes the specified item from heap.
// Note, this API suppose the specified item exists in heap.
// Otherwise, any data will be removed from heap accidently.
func (h *Heap) Remove(item HeapItem) {
	heap.Remove(h, item.getHeapIndex())
}
