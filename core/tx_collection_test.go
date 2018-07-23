/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/core/types"
)

func Test_txCollection_add(t *testing.T) {
	collection := newTxCollection()

	// add item
	tx1 := newTestPoolTx(t, 10, 5)
	collection.add(tx1)
	assert.Equal(t, collection.list(), []*types.Transaction{tx1.Transaction})

	// add bigger nonce item
	tx2 := newTestPoolTx(t, 10, 6)
	collection.add(tx2)
	assert.Equal(t, collection.list(), []*types.Transaction{tx1.Transaction, tx2.Transaction})

	// add smaller nonce item
	tx3 := newTestPoolTx(t, 10, 4)
	collection.add(tx3)
	assert.Equal(t, collection.list(), []*types.Transaction{tx3.Transaction, tx1.Transaction, tx2.Transaction})
}

func Test_txCollection_update(t *testing.T) {
	collection := newTxCollection()

	collection.add(newTestPoolTx(t, 1, 5))
	collection.add(newTestPoolTx(t, 2, 5))
	collection.add(newTestPoolTx(t, 3, 5))

	txs := collection.list()
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, txs[0].Data.Amount.Int64(), int64(3))
}

func Test_txCollection_get(t *testing.T) {
	collection := newTxCollection()

	poolTx := newTestPoolTx(t, 10, 5)
	collection.add(poolTx)

	assert.Equal(t, collection.get(5), poolTx)
	assert.Equal(t, collection.get(6) == nil, true)
}

func Test_txCollection_Remove(t *testing.T) {
	collection := newTxCollection()

	assert.Equal(t, len(collection.list()), 0)
	assert.Equal(t, collection.remove(2), false)

	tx1, tx2, tx3 := newTestPoolTx(t, 1, 3), newTestPoolTx(t, 1, 4), newTestPoolTx(t, 1, 2)
	collection.add(tx1)
	collection.add(tx2)
	collection.add(tx3)
	assert.Equal(t, collection.list(), []*types.Transaction{tx3.Transaction, tx1.Transaction, tx2.Transaction})

	assert.Equal(t, collection.remove(3), true)
	assert.Equal(t, collection.list(), []*types.Transaction{tx3.Transaction, tx2.Transaction})
}

func Test_txCollection_len(t *testing.T) {
	collection := newTxCollection()

	for nonce := uint64(5); nonce < uint64(13); nonce++ {
		collection.add(newTestPoolTx(t, 10, nonce))
	}

	assert.Equal(t, collection.len(), 8)
}

func Test_txCollection_peek(t *testing.T) {
	collection := newTxCollection()

	// add item
	tx1 := newTestPoolTx(t, 10, 5)
	collection.add(tx1)
	assert.Equal(t, collection.peek(), tx1)

	// add bigger nonce item
	tx2 := newTestPoolTx(t, 10, 6)
	collection.add(tx2)
	assert.Equal(t, collection.peek(), tx1)

	// add smaller nonce item
	tx3 := newTestPoolTx(t, 10, 4)
	collection.add(tx3)
	assert.Equal(t, collection.peek(), tx3)
}

func Test_txCollection_pop(t *testing.T) {
	collection := newTxCollection()

	tx1, tx2, tx3 := newTestPoolTx(t, 1, 3), newTestPoolTx(t, 1, 4), newTestPoolTx(t, 1, 2)
	collection.add(tx1)
	collection.add(tx2)
	collection.add(tx3)

	// pop tx3, nonce = 2
	assert.Equal(t, collection.pop(), tx3)
	assert.Equal(t, collection.len(), 2)

	// pop tx1, nonce = 3
	assert.Equal(t, collection.pop(), tx1)
	assert.Equal(t, collection.len(), 1)

	// pop tx2, nonce = 4
	assert.Equal(t, collection.pop(), tx2)
	assert.Equal(t, collection.len(), 0)
}
