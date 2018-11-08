/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/event"
	"github.com/stretchr/testify/assert"
)

func Test_TxPool_NewTxPool(t *testing.T) {
	chain := &TestBlockChain{}
	ob := newOdrBackend(chain.GetStore(), 1)
	txPool := newTxPool(chain, ob, event.NewEventManager())
	defer txPool.stop()

	assert.NotNil(t, txPool)
	assert.Equal(t, txPool.chain, chain)
	assert.Equal(t, txPool.odrBackend, ob)
	assert.Equal(t, len(txPool.pendingTxs), 0)
	assert.Equal(t, len(txPool.minedBlocks), 0)
	assert.Equal(t, len(txPool.headerCh), 0)
	assert.Nil(t, txPool.currentHeader)
	assert.NotNil(t, txPool.log)
}

func Test_TxPool_AddTransaction(t *testing.T) {
	chain := &TestBlockChain{}
	ob := newOdrBackend(chain.GetStore(), 1)
	txPool := newTxPool(chain, ob, event.NewEventManager())
	defer txPool.stop()

	// case 1: tx is nil
	err := txPool.AddTransaction(nil)
	assert.Nil(t, err)

	// case 2: tx is invalid
	tx := newTestTx(10, 1, 1, true)
	tx.Hash = common.EmptyHash
	err = txPool.AddTransaction(tx)
	assert.True(t, errors.IsOrContains(err, types.ErrHashMismatch))
}

func Test_TxPool_GetTransactions(t *testing.T) {
	chain := &TestBlockChain{}
	ob := newOdrBackend(chain.GetStore(), 1)
	txPool := newTxPool(chain, ob, event.NewEventManager())
	defer txPool.stop()

	pooledTx := txPool.GetTransaction(common.EmptyHash)
	assert.Nil(t, pooledTx)

	pooledTxs := txPool.GetTransactions(false, false)
	assert.Nil(t, pooledTxs)

	pooledTxs = txPool.GetTransactions(false, true)
	assert.Nil(t, pooledTxs)

	txCount := txPool.GetPendingTxCount()
	assert.Equal(t, txCount, 0)

	newTx := newTestTx(10, 1, 1, true)
	txPool.pendingTxs[newTx.Hash] = newTx

	pooledTx = txPool.GetTransaction(newTx.Hash)
	assert.Equal(t, pooledTx, newTx)

	pooledTx = txPool.GetTransaction(common.EmptyHash)
	assert.Nil(t, pooledTx)

	pooledTxs = txPool.GetTransactions(false, false)
	assert.Nil(t, pooledTxs)

	pooledTxs = txPool.GetTransactions(false, true)
	assert.NotNil(t, pooledTxs)
	assert.Equal(t, len(pooledTxs), 1)

	txCount = txPool.GetPendingTxCount()
	assert.Equal(t, txCount, 1)
}
