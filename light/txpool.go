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
	pending    map[common.Hash]*types.Transaction
	log        *log.SeeleLog
}

func newTxPool(chain BlockChain, odrBackend *odrBackend) *txPool {
	return &txPool{
		chain:      chain,
		odrBackend: odrBackend,
		pending:    make(map[common.Hash]*types.Transaction),
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

	if pool.pending[tx.Hash] != nil {
		return fmt.Errorf("Transaction already exists, hash is %v", tx.Hash.ToHex())
	}

	request := &odrAddTx{Tx: *tx}
	if err := pool.odrBackend.sendRequest(request); err != nil {
		return fmt.Errorf("Failed to send request to peers, %v", err.Error())
	}

	if err := request.getError(); err != nil {
		return err
	}

	pool.pending[tx.Hash] = tx

	return nil
}
