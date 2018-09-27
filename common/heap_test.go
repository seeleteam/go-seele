/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testHeapItem struct {
	BaseHeapItem
	Num int
}

func Test_Heap(t *testing.T) {
	h := NewHeap(func(i, j HeapItem) bool {
		return i.(*testHeapItem).Num < j.(*testHeapItem).Num
	})

	// empty heap
	assert.Equal(t, 0, h.Len())

	// 1 item
	heap.Push(h, &testHeapItem{Num: 18})
	assert.Equal(t, 1, h.Len())
	assert.Equal(t, &testHeapItem{Num: 18}, h.Peek())

	// 2 items - min
	heap.Push(h, &testHeapItem{Num: 16})
	assert.Equal(t, 2, h.Len())
	assert.Equal(t, &testHeapItem{Num: 16}, h.Peek())

	// 3 items - not min
	heap.Push(h, &testHeapItem{Num: 17})
	assert.Equal(t, 3, h.Len())
	assert.Equal(t, &testHeapItem{Num: 16}, h.Peek())

	// pop
	assert.Equal(t, &testHeapItem{BaseHeapItem: BaseHeapItem{2}, Num: 16}, heap.Pop(h))
	assert.Equal(t, &testHeapItem{BaseHeapItem: BaseHeapItem{1}, Num: 17}, heap.Pop(h))
	assert.Equal(t, &testHeapItem{BaseHeapItem: BaseHeapItem{0}, Num: 18}, heap.Pop(h))
	assert.Equal(t, 0, h.Len())
}
