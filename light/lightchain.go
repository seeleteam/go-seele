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
	log *log.SeeleLog
}

func newLightChain(bcStore store.BlockchainStore, lightDB database.Database) (*LightChain, error) {
	//todo
	return nil, nil
}

func (bc *LightChain) CurrentBlock() *types.Block {
	return nil
}

func (bc *LightChain) GetStore() store.BlockchainStore {
	return nil
}
