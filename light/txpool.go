/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"errors"
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

	statedb, err := pool.chain.GetCurrentState()
	if err != nil {
		return fmt.Errorf("Failed to get current state from chain, %v", err.Error())
	}

	if err := tx.Validate(statedb); err != nil {
		return fmt.Errorf("Failed to validate tx, %v", err.Error())
	}

	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if pool.pending[tx.Hash] != nil {
		return fmt.Errorf("transaction already exists, hash is %v", tx.Hash.ToHex())
	}

	pool.pending[tx.Hash] = tx

	request := &odrAddTx{
		Tx: *tx,
	}

	if err := pool.odrBackend.sendRequest(request); err != nil {
		return fmt.Errorf("Failed to send request to peers, %v", err.Error())
	}

	if len(request.Error) > 0 {
		return errors.New(request.Error)
	}

	return nil
}
