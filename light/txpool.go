/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

type LightPool struct {
	odrBackend *odrBackend
	log        *log.SeeleLog
}

func newLightPool(chain BlockChain, odrBackend *odrBackend) (*LightPool, error) {
	pool := &LightPool{
		odrBackend: odrBackend,
	}

	return pool, nil
}

// AddTransaction adds a single transaction into the pool if it is valid and returns nil.
// Otherwise, return the concrete error.
func (pool *LightPool) AddTransaction(tx *types.Transaction) error {
	return nil
}

// GetTransaction returns a transaction if it is contained in the pool and nil otherwise.
func (pool *LightPool) GetTransaction(txHash common.Hash) *types.Transaction {
	return nil
}

// GetTransactions return the transactions in the transaction pool.
func (pool *LightPool) GetTransactions(processing, pending bool) []*types.Transaction {
	return nil
}

// GetPendingTxCount return the total number of pending transactions in the transaction pool.
func (pool *LightPool) GetPendingTxCount() int {
	return 0
}
