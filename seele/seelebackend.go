/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database/leveldb"
)

// main backend of seele interface
type SeeleBackend interface {
	TxPool() *core.TransactionPool
	BlockChain() *core.Blockchain
	ApplyTransaction(coinbase common.Address, tx *types.Transaction) error
}

// fake impl of SeeleBackend
type SeeleBackendImpl struct {
	txPool 		*core.TransactionPool
	chain 		*core.Blockchain
}

func NewSeeleBackendImpl(dbPath string) *SeeleBackendImpl {
	seele := &SeeleBackendImpl {
		txPool:		core.NewTransactionPool(*core.DefaultTxPoolConfig()),
	}
	
	db, err := leveldb.NewLevelDB(dbPath)
	if err != nil {
		fmt.Println("New leveldb instance failed")
		return nil
	}

	seele.chain, err = core.NewBlockchain(store.NewBlockchainDatabase(db))
	if err != nil {
		fmt.Println("New blockchain instance failed")
		return nil
	}

	return seele
}

func (seele *SeeleBackendImpl) TxPool() *core.TransactionPool {
	return seele.txPool
}

func (seele *SeeleBackendImpl) BlockChain() *core.Blockchain {
	return seele.chain
}

func (seele *SeeleBackendImpl) ApplyTransaction(coinbase common.Address, tx *types.Transaction) error {
	return nil
}
