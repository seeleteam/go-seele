/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/core/store"

	"github.com/seeleteam/go-seele/database"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
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

func Test_HeaderChain_WriteHeader_InvalidParentHash(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	hc := newTestHeaderChain(db)

	newHeader := &types.BlockHeader{
		PreviousBlockHash: common.EmptyHash,
	}

	err := hc.WriteHeader(newHeader)
	assert.Equal(t, err, ErrHeaderChainInvalidParentHash)
}

func Test_HeaderChain_WriteHeader_InvalidHeight(t *testing.T) {
	db, dispose := newTestDatabase()
	defer dispose()

	hc := newTestHeaderChain(db)

	newHeader := &types.BlockHeader{
		PreviousBlockHash: hc.genesisHeader.Hash(),
		Height:            2,
	}

	err := hc.WriteHeader(newHeader)
	assert.Equal(t, err, ErrHeaderChainInvalidHeight)
}

func Test_HeaderChain_WriteHeader_ValidHeader(t *testing.T) {
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

	// Ensure current header info updated
	assert.Equal(t, err, error(nil))
	assert.Equal(t, hc.currentHeaderHash, newHeader.Hash())
	assert.Equal(t, hc.currentHeader, newHeader)

	// Ensure store updated.
	headHash, _ := hc.bcStore.GetHeadBlockHash()
	headHeader, _ := hc.bcStore.GetBlockHeader(headHash)
	assert.Equal(t, headHeader, newHeader)
	td, _ := hc.bcStore.GetBlockTotalDifficulty(headHash)
	assert.Equal(t, td, new(big.Int).Add(hc.genesisHeader.Difficulty, newHeader.Difficulty))
}
