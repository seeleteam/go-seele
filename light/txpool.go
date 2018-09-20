/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"fmt"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

type txPool struct {
	mutex      sync.RWMutex
	chain      BlockChain
	odrBackend *odrBackend
	pendingTxs map[common.Hash]*types.Transaction   // map[txHash]tx
	minedTxs   map[common.Hash][]*types.Transaction // map[blockHash][]tx
	log        *log.SeeleLog
}

func newTxPool(chain BlockChain, odrBackend *odrBackend) *txPool {
	return &txPool{
		chain:      chain,
		odrBackend: odrBackend,
		pendingTxs: make(map[common.Hash]*types.Transaction),
		minedTxs:   make(map[common.Hash][]*types.Transaction),
		log:        log.GetLogger("lightTxPool"),
	}
}

func (pool *txPool) AddTransaction(tx *types.Transaction) error {
	if tx == nil {
		return nil
	}

	if err := tx.ValidateWithoutState(true, false); err != nil {
		return err
	}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if pool.pendingTxs[tx.Hash] != nil {
		return fmt.Errorf("Transaction already exists, hash is %v", tx.Hash.ToHex())
	}

	request := &odrAddTx{Tx: *tx}
	if err := pool.odrBackend.sendRequest(request); err != nil {
		return fmt.Errorf("Failed to send request to peers, %v", err.Error())
	}

	if err := request.getError(); err != nil {
		return err
	}

	pool.pendingTxs[tx.Hash] = tx

	return nil
}

// GetTransaction returns a transaction if it is contained in the pool and nil otherwise.
func (pool *txPool) GetTransaction(txHash common.Hash) *types.Transaction {
	return pool.pendingTxs[txHash]
}

// GetTransactions return the transactions in the transaction pool.
func (pool *txPool) GetTransactions(processing, pending bool) []*types.Transaction {
	if !pending || len(pool.pendingTxs) == 0 {
		return nil
	}

	txs := make([]*types.Transaction, len(pool.pendingTxs))
	i := 0

	for _, tx := range pool.pendingTxs {
		txs[i] = tx
		i++
	}

	return txs
}

// GetPendingTxCount return the total number of pending transactions in the transaction pool.
func (pool *txPool) GetPendingTxCount() int {
	return len(pool.pendingTxs)
}
