/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
)

func testHeaderChain(t *testing.T, ut func(*HeaderChain)) {
	testBlockchainDatabase(func(bcStore store.BlockchainStore) {
		genesis := DefaultGenesis(bcStore)
		if err := genesis.Initialize(); err != nil {
			t.Fatal()
		}

		hc, err := NewHeaderChain(bcStore)
		if err != nil {
			t.Fatal()
		}

		ut(hc)
	})
}

func Test_HeaderChain_WriteHeader_InvalidParentHash(t *testing.T) {
	testHeaderChain(t, func(hc *HeaderChain) {
		newHeader := &types.BlockHeader{
			PreviousBlockHash: common.EmptyHash,
		}

		err := hc.WriteHeader(newHeader)
		assert.Equal(t, err, ErrHeaderChainInvalidParentHash)
	})
}

func Test_HeaderChain_WriteHeader_InvalidHeight(t *testing.T) {
	testHeaderChain(t, func(hc *HeaderChain) {
		newHeader := &types.BlockHeader{
			PreviousBlockHash: hc.genesisHeader.Hash(),
			Height:            2,
		}

		err := hc.WriteHeader(newHeader)
		assert.Equal(t, err, ErrHeaderChainInvalidHeight)
	})
}

func Test_HeaderChain_WriteHeader_ValidHeader(t *testing.T) {
	testHeaderChain(t, func(hc *HeaderChain) {
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
	})
}
