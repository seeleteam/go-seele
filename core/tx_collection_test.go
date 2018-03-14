/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_txCollection_add(t *testing.T) {
	collection := newTxCollection()
	tx := newTestTx(t, 10, 5)
	collection.add(tx)

	txs := collection.getTxsOrderByNonceAsc()
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, txs[0], tx)
}

func Test_txCollection_add_txsWithSameNonce(t *testing.T) {
	collection := newTxCollection()
	collection.add(newTestTx(t, 1, 5))
	collection.add(newTestTx(t, 2, 5))
	collection.add(newTestTx(t, 3, 5))

	txs := collection.getTxsOrderByNonceAsc()
	assert.Equal(t, len(txs), 1)
	assert.Equal(t, txs[0].Data.Amount.Int64(), int64(3))
}

func Test_txCollection_getTxsOrderByNonceAsc(t *testing.T) {
	collection := newTxCollection()
	collection.add(newTestTx(t, 3, 9))
	collection.add(newTestTx(t, 1, 5))
	collection.add(newTestTx(t, 2, 7))

	txs := collection.getTxsOrderByNonceAsc()
	assert.Equal(t, len(txs), 3)
	assert.Equal(t, txs[0].Data.Amount.Int64(), int64(1))
	assert.Equal(t, txs[1].Data.Amount.Int64(), int64(2))
	assert.Equal(t, txs[2].Data.Amount.Int64(), int64(3))
}
