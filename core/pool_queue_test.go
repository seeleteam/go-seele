/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/stretchr/testify/assert"
)

func uintToAddress(i uint64) common.Address {
	return common.BigToAddress(new(big.Int).SetUint64(i))
}

func newMockPooledTx(fromAddr, price, nonce uint64) *poolItem {
	return &poolItem{
		poolObject: &types.Transaction{
			Data: types.TransactionData{
				From:         uintToAddress(fromAddr),
				GasPrice:     new(big.Int).SetUint64(price),
				AccountNonce: nonce,
			},
		},
		timestamp: time.Now(),
	}
}

func Test_PendingQueue_add(t *testing.T) {
	q := newPendingQueue()

	q.add(newMockPooledTx(1, 2, 5))
	assert.Equal(t, q.count(), 1)

	q.add(newMockPooledTx(2, 2, 5))
	assert.Equal(t, q.count(), 2)
}

func Test_PendingQueue_update(t *testing.T) {
	q := newPendingQueue()

	tx1 := newMockPooledTx(1, 2, 5)
	q.add(tx1)
	assert.Equal(t, 1, q.count())
	assert.Equal(t, tx1, q.peek().peek())

	// update tx with higher price
	tx2 := newMockPooledTx(1, 3, 5)
	q.add(tx2)
	assert.Equal(t, 1, q.count())
	assert.Equal(t, tx2, q.peek().peek())
}

func Test_PendingQueue_get(t *testing.T) {
	q := newPendingQueue()

	tx := newMockPooledTx(1, 2, 5)
	q.add(tx)

	assert.Equal(t, q.get(uintToAddress(1), 5), tx)
	assert.Equal(t, q.get(uintToAddress(2), 5) == nil, true)
	assert.Equal(t, q.get(uintToAddress(1), 6) == nil, true)
}

func Test_PendingQueue_remove(t *testing.T) {
	q := newPendingQueue()

	q.add(newMockPooledTx(1, 2, 5))
	q.add(newMockPooledTx(1, 2, 6))
	q.add(newMockPooledTx(2, 2, 5))
	q.add(newMockPooledTx(2, 2, 6))
	assert.Equal(t, q.count(), 4)

	// remove with invalid address
	q.remove(uintToAddress(3), 5)
	assert.Equal(t, q.count(), 4)

	// remove with invalid nonce
	q.remove(uintToAddress(1), 7)
	assert.Equal(t, q.count(), 4)

	// remove with valid address and nonce
	q.remove(uintToAddress(1), 5)
	assert.Equal(t, q.count(), 3)
	q.remove(uintToAddress(1), 6)
	assert.Equal(t, q.count(), 2)
	q.remove(uintToAddress(2), 5)
	assert.Equal(t, q.count(), 1)
	q.remove(uintToAddress(2), 6)
	assert.Equal(t, q.count(), 0)
}

func Test_PendingQueue_peek(t *testing.T) {
	q := newPendingQueue()

	// first account with price 5
	tx1 := newMockPooledTx(1, 5, 1)
	q.add(tx1)
	assert.Equal(t, q.peek().peek(), tx1)

	// insert tx with less price 4
	tx2 := newMockPooledTx(2, 4, 1)
	q.add(tx2)
	assert.Equal(t, q.peek().peek(), tx1)

	// insert tx with more price 6
	tx3 := newMockPooledTx(3, 6, 1)
	q.add(tx3)
	assert.Equal(t, q.peek().peek(), tx3)

	// insert tx with same price 6, but latest time
	tx4 := newMockPooledTx(4, 6, 1)
	q.add(tx4)
	assert.Equal(t, q.peek().peek(), tx3)

	// insert tx with same price 6, but older time
	tx5 := newMockPooledTx(5, 6, 1)
	tx5.timestamp = tx3.timestamp.Add(-3 * time.Second)
	q.add(tx5)
	assert.Equal(t, q.peek().peek(), tx5)
}

func Test_PendingQueue_popN(t *testing.T) {
	q := newPendingQueue()

	accountTxs := map[uint][]*poolItem{
		1: []*poolItem{newMockPooledTx(1, 5, 1), newMockPooledTx(1, 20, 2)},
		2: []*poolItem{newMockPooledTx(2, 7, 1), newMockPooledTx(2, 15, 2)},
		3: []*poolItem{newMockPooledTx(3, 9, 1), newMockPooledTx(3, 6, 2)},
	}

	for _, txs := range accountTxs {
		for _, tx := range txs {
			q.add(tx)
		}
	}

	assert.Equal(t, q.popN(-1) == nil, true)
	assert.Equal(t, q.popN(0) == nil, true)

	assert.Equal(t, q.popN(100), []poolObject{
		accountTxs[3][0].poolObject, // pop price 9
		accountTxs[2][0].poolObject, // pop price 7
		accountTxs[2][1].poolObject, // pop price 15
		accountTxs[3][1].poolObject, // pop price 6
		accountTxs[1][0].poolObject, // pop price 5
		accountTxs[1][1].poolObject, // pop price 20
	})

	assert.Equal(t, q.popN(1) == nil, true)
}

func Test_pendingQueue_pop(t *testing.T) {
	q := newPendingQueue()

	ptx1 := newMockPooledTx(1, 3, 5)
	q.add(ptx1)

	ptx2 := newMockPooledTx(2, 2, 1)
	q.add(ptx2)

	ptx3 := newMockPooledTx(2, 5, 6)
	q.add(ptx3)

	ptx4 := newMockPooledTx(2, 1, 2)
	q.add(ptx4)

	assert.Equal(t, q.pop(), ptx1.poolObject)
	assert.Equal(t, q.pop(), ptx2.poolObject)
	assert.Equal(t, q.pop(), ptx4.poolObject)
	assert.Equal(t, q.pop(), ptx3.poolObject)
}

func Test_pendingQueue_discard_emptyQueue(t *testing.T) {
	q := newPendingQueue()
	assert.Nil(t, q.discard(big.NewInt(10)))
}

func Test_pendingQueue_discard_oneAccount(t *testing.T) {
	q := newPendingQueue()

	// add a tx in queue, price is 38
	ptx1 := newMockPooledTx(1, 38, 1)
	q.add(ptx1)
	assert.False(t, q.empty())

	// failed to discard with lower or same price
	assert.Nil(t, q.discard(big.NewInt(37)))
	assert.Nil(t, q.discard(big.NewInt(38)))

	// succeed to discard with higher price
	assert.Equal(t, ptx1, q.discard(big.NewInt(39)).peek())
	assert.True(t, q.empty())
	assert.Equal(t, 0, len(q.txs))
	assert.Equal(t, 0, q.bestHeap.Len())
	assert.Equal(t, 0, q.worstHeap.Len())
}

func Test_pendingQueue_discard_cmp(t *testing.T) {
	q := newPendingQueue()

	// tx1
	ptx1 := newMockPooledTx(1, 38, 1)
	q.add(ptx1)

	// tx2: lower price, discard tx2, left tx1
	ptx2 := newMockPooledTx(2, 37, 1)
	q.add(ptx2)
	assert.Equal(t, ptx2, q.discard(big.NewInt(100)).peek())
	assert.Equal(t, ptx1, q.peek().peek())

	// tx3: higher price, discard tx1, left tx3
	ptx3 := newMockPooledTx(3, 40, 1)
	q.add(ptx3)
	assert.Equal(t, ptx1, q.discard(big.NewInt(100)).peek())
	assert.Equal(t, ptx3, q.peek().peek())

	// tx4: same price with later timestamp, discard tx4, left tx3
	ptx4 := newMockPooledTx(4, 40, 1)
	ptx4.timestamp = ptx3.timestamp.Add(time.Second)
	q.add(ptx4)
	assert.Equal(t, ptx4, q.discard(big.NewInt(100)).peek())
	assert.Equal(t, ptx3, q.peek().peek())

	// tx5: same price with earlier timestamp, discard tx3, left tx5
	ptx5 := newMockPooledTx(5, 40, 1)
	ptx5.timestamp = ptx3.timestamp.Add(-time.Second)
	q.add(ptx5)
	assert.Equal(t, ptx3, q.discard(big.NewInt(100)).peek())
	assert.Equal(t, ptx5, q.peek().peek())

	// tx6: same price and timestamp, discard tx6, left tx5 (LIFO)
	ptx6 := newMockPooledTx(6, 40, 1)
	ptx6.timestamp = ptx5.timestamp
	q.add(ptx6)
	assert.Equal(t, ptx6, q.discard(big.NewInt(100)).peek())
	assert.Equal(t, ptx5, q.peek().peek())
}

func Test_pendingQueue_list(t *testing.T) {
	q := newPendingQueue()

	ptx1 := newMockPooledTx(1, 3, 5)
	q.add(ptx1)

	ptx2 := newMockPooledTx(2, 2, 1)
	q.add(ptx2)

	ptx3 := newMockPooledTx(2, 5, 6)
	q.add(ptx3)

	ptx4 := newMockPooledTx(2, 1, 2)
	q.add(ptx4)

	txs := q.list()
	assert.Equal(t, len(txs), 4)
}

func Benchmark_PendingQueue_popN(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		txs := prepareTxs(DefaultTxPoolConfig().Capacity, 3)
		q := preparePendingQueue(txs)
		b.StartTimer()

		if txs := q.popN(BlockTransactionNumberLimit); len(txs) != BlockTransactionNumberLimit {
			b.Fatal()
		}
	}
}

func preparePendingQueue(txs map[common.Address][]*poolItem) *pendingQueue {
	q := newPendingQueue()

	for _, nonceSortedTxs := range txs {
		for _, tx := range nonceSortedTxs {
			q.add(tx)
		}
	}

	return q
}

func prepareTxs(numAccounts, numTxsPerAccount uint) map[common.Address][]*poolItem {
	txs := make(map[common.Address][]*poolItem)

	for i := uint(1); i <= numAccounts; i++ {
		from := common.BigToAddress(big.NewInt(int64(i)))

		accountTxs := make([]*poolItem, 0, numTxsPerAccount)
		for j := uint(1); j <= numTxsPerAccount; j++ {
			tx := &types.Transaction{
				Data: types.TransactionData{
					From:         from,
					GasPrice:     big.NewInt(int64(rand.Intn(10000))),
					AccountNonce: uint64(j),
				},
			}
			ptx := newPooledItem(tx)
			tx.Data.Timestamp = uint64(ptx.timestamp.UnixNano())
			accountTxs = append(accountTxs, ptx)
		}

		txs[from] = accountTxs
	}

	return txs
}
