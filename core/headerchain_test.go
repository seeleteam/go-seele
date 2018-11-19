/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func newTestHeaderChain(db database.Database) *HeaderChain {
	bcStore := store.NewBlockchainDatabase(db)

	genesis := GetGenesis(&GenesisInfo{})
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	}

	hc, err := NewHeaderChain(bcStore)
	if err != nil {
		panic(err)
	}

	return hc
}

func Test_HeaderChain_NewHeaderChain(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bcStore := store.NewBlockchainDatabase(db)
	_, err := NewHeaderChain(bcStore)
	assert.Equal(t, err != nil, true)

	genesis := GetGenesis(&GenesisInfo{})
	genesis.InitializeAndValidate(bcStore, db)
	hc, err := NewHeaderChain(bcStore)
	assert.Equal(t, err == nil, true)
	assert.Equal(t, hc != nil, true)
}

func Test_HeaderChain_WriteHeader(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
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

	// ensure header is cloned
	newHeader.PreviousBlockHash = common.StringToHash("newHash")
	assert.Equal(t, hc.currentHeader != newHeader, true)
}
