/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/log"
)

type LightChain struct {
	odrBackend *odrBackend
	log        *log.SeeleLog
}

func newLightChain(bcStore store.BlockchainStore, lightDB database.Database, odrBackend *odrBackend) (*LightChain, error) {
	chain := &LightChain{
		odrBackend: odrBackend,
	}
	return chain, nil
}

func (bc *LightChain) CurrentBlock() *types.Block {
	return nil
}

func (bc *LightChain) GetCurrentState() (*state.Statedb, error) {
	return nil, nil
}

func (bc *LightChain) GetState(root common.Hash) (*state.Statedb, error) {
	return nil, nil
}

func (bc *LightChain) GetStore() store.BlockchainStore {
	return nil
}

func (bc *LightChain) WriteHeader(*types.BlockHeader) error {
	return nil
}

// ApplyTransaction applies a transaction, changes corresponding statedb and generates its receipt
func (bc *LightChain) ApplyTransaction(tx *types.Transaction, txIndex int, coinbase common.Address, statedb *state.Statedb,
	blockHeader *types.BlockHeader) (*types.Receipt, error) {
	return nil, nil
}

func (bc *LightChain) GetCurrentStateNonce() (uint64, error) {
	return 0, nil
}

func (bc *LightChain) GetCurrentStateBalance(account common.Address) (*big.Int, error) {
	return nil, nil
}
