/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"testing"

	"github.com/seeleteam/go-seele/core/types"
	"github.com/stretchr/testify/assert"
)

func Test_TransactionPool_Add_InvalidTx(t *testing.T) {
	pool, chain := newTestPool(DefaultTxPoolConfig())
	defer chain.dispose()

	txpool := &TransactionPool{pool}

	poolTx := newTestPoolTx(t, 30, 100)
	chain.addAccount(poolTx.Account(), 20, 100)

	// Change the amount in tx.
	err := txpool.AddTransaction(poolTx.poolObject.(*types.Transaction))

	if err == nil {
		t.Fatal("The error is nil when add invalid tx to pool.")
	}

	// add nil tx
	err = txpool.AddTransaction(nil)
	assert.Equal(t, err, error(nil))
}
