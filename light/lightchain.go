/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
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

func (bc *LightChain) GetStore() store.BlockchainStore {
	return nil
}
