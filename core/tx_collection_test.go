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
	poolTx := newTestPoolTx(t, 10, 5)
	collection.add(poolTx)

	txs := collection.getTxsOrderByNonceAsc(ALL)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, txs[0], poolTx.transaction)
}

func Test_txCollection_add_txsWithSameNonce(t *testing.T) {
	collection := newTxCollection()
	collection.add(newTestPoolTx(t, 1, 5))
	collection.add(newTestPoolTx(t, 2, 5))
	collection.add(newTestPoolTx(t, 3, 5))

	txs := collection.getTxsOrderByNonceAsc(ALL)
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, txs[0].Data.Amount.Int64(), int64(3))
}

func Test_txCollection_Remove(t *testing.T) {
	collection := newTxCollection()
	poolTx := newTestPoolTx(t, 1, 2)
	collection.add(poolTx)

	assert.Equal(t, collection.count(ALL), 1)
	collection.remove(poolTx.transaction.Data.AccountNonce)
	assert.Equal(t, collection.count(ALL), 0)
}

func Test_txCollection_getTxsOrderByNonceAsc(t *testing.T) {
	collection := newTxCollection()
	collection.add(newTestPoolTx(t, 3, 9))
	collection.add(newTestPoolTx(t, 1, 5))
	collection.add(newTestPoolTx(t, 2, 7))

	txs := collection.getTxsOrderByNonceAsc(ALL)
	assert.Equal(t, len(txs), 3)
	assert.Equal(t, txs[0].Data.Amount.Int64(), int64(1))
	assert.Equal(t, txs[1].Data.Amount.Int64(), int64(2))
	assert.Equal(t, txs[2].Data.Amount.Int64(), int64(3))
}

func Test_txCollection_getPendingTxsOrderByNonceAsc(t *testing.T) {
	collection := newTxCollection()
	poolTx1 := newTestPoolTx(t, 3, 9)
	poolTx2 := newTestPoolTx(t, 1, 5)
	poolTx3 := newTestPoolTx(t, 2, 7)
	poolTx2.txStatus = PROCESSING
	collection.add(poolTx1)
	collection.add(poolTx2)
	collection.add(poolTx3)

	txs := collection.getTxsOrderByNonceAsc(PENDING)
	assert.Equal(t, len(txs), 2)
	assert.Equal(t, txs[0].Data.Amount.Int64(), int64(2))
	assert.Equal(t, txs[1].Data.Amount.Int64(), int64(3))
}

func Test_txCollection_count(t *testing.T) {
	collection := newTxCollection()
	poolTx1 := newTestPoolTx(t, 3, 9)
	poolTx2 := newTestPoolTx(t, 1, 5)
	poolTx3 := newTestPoolTx(t, 2, 7)
	poolTx2.txStatus = PROCESSING
	collection.add(poolTx1)
	collection.add(poolTx2)
	collection.add(poolTx3)

	account := collection.count(ALL)
	assert.Equal(t, account, 3)
}

func Test_txCollection_countPendingTxs(t *testing.T) {
	collection := newTxCollection()
	poolTx1 := newTestPoolTx(t, 3, 9)
	poolTx2 := newTestPoolTx(t, 1, 5)
	poolTx3 := newTestPoolTx(t, 2, 7)
	poolTx2.txStatus = PROCESSING
	collection.add(poolTx1)
	collection.add(poolTx2)
	collection.add(poolTx3)

	account := collection.count(PENDING)
	assert.Equal(t, account, 2)
}

func Test_txCollection_findTx(t *testing.T) {
	collection := newTxCollection()
	poolTx1 := newTestPoolTx(t, 3, 9)
	poolTx2 := newTestPoolTx(t, 1, 5)
	poolTx3 := newTestPoolTx(t, 2, 7)
	poolTx2.txStatus = PROCESSING
	collection.add(poolTx1)
	collection.add(poolTx2)
	collection.add(poolTx3)

	tx := collection.findTx(9, ALL)
	assert.Equal(t, tx, poolTx1.transaction)
	tx = collection.findTx(5, ALL)
	assert.Equal(t, tx, poolTx2.transaction)
	tx = collection.findTx(7, ALL)
	assert.Equal(t, tx, poolTx3.transaction)

}

func Test_txCollection_findPendingTx(t *testing.T) {
	collection := newTxCollection()
	poolTx1 := newTestPoolTx(t, 3, 9)
	poolTx2 := newTestPoolTx(t, 1, 5)
	poolTx3 := newTestPoolTx(t, 2, 7)
	poolTx2.txStatus = PROCESSING
	collection.add(poolTx1)
	collection.add(poolTx2)
	collection.add(poolTx3)

	tx := collection.findTx(9, PENDING)
	assert.Equal(t, tx, poolTx1.transaction)
	tx = collection.findTx(5, PENDING)
	assert.Equal(t, tx, (*types.Transaction)(nil))
	tx = collection.findTx(10, PENDING)
	assert.Equal(t, tx, (*types.Transaction)(nil))
	tx = collection.findTx(7, PENDING)
	assert.Equal(t, tx, poolTx3.transaction)
}
