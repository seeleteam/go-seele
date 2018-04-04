/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
)

func newTestHeaderChain(db database.Database) *HeaderChain {
	bcStore := store.NewBlockchainDatabase(db)

	genesis := DefaultGenesis(bcStore)
	if err := genesis.Initialize(db); err != nil {
		panic(err)
	}

	hc, err := NewHeaderChain(bcStore)
	if err != nil {
		panic(err)
	}

	return hc
}

func Test_HeaderChain_WriteHeader(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	hc := newTestHeaderChain(db)
	newHeader := &types.BlockHeader{
		PreviousBlockHash: hc.genesisHeader.Hash(),
		Height:            1,
		Difficulty:        big.NewInt(78),
		CreateTimestamp:   big.NewInt(1),
	}

	err := hc.WriteHeader(newHeader)
	assert.Equal(t, err, nil)
	assert.Equal(t, hc.currentHeaderHash, newHeader.Hash())
	assert.Equal(t, hc.currentHeader, newHeader)
}
