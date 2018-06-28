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
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func Test_Genesis_GetGenesis(t *testing.T) {
	genesis1 := GetGenesis(GenesisInfo{})
	genesis2 := GetGenesis(GenesisInfo{})

	assert.Equal(t, genesis1.header, genesis2.header)

	addr := crypto.MustGenerateRandomAddress()
	accounts := make(map[common.Address]*big.Int)
	accounts[*addr] = big.NewInt(10)
	genesis3 := GetGenesis(GenesisInfo{accounts, 1, 0})
	if genesis3.header.StateHash == common.EmptyHash {
		panic("genesis3 state hash should not equal to empty hash")
	}

	if genesis3.header == genesis2.header {
		panic("genesis3 should not equal to genesis2")
	}
}

func Test_Genesis_Init_DefaultGenesis(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bcStore := store.NewBlockchainDatabase(db)

	genesis := GetGenesis(GenesisInfo{})
	genesisHash := genesis.header.Hash()

	err := genesis.InitializeAndValidate(bcStore, db)
	if err != nil {
		panic(err)
	}

	hash, err := bcStore.GetBlockHash(genesisBlockHeight)
	assert.Equal(t, err, error(nil))
	assert.Equal(t, hash, genesisHash)

	headHash, err := bcStore.GetHeadBlockHash()
	assert.Equal(t, err, error(nil))
	assert.Equal(t, headHash, genesisHash)

	_, err = state.NewStatedb(genesis.header.StateHash, db)
	assert.Equal(t, err, error(nil))
}

func Test_Genesis_Init_GenesisMismatch(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bcStore := store.NewBlockchainDatabase(db)

	header := GetGenesis(GenesisInfo{}).header.Clone()
	header.Nonce = 38
	bcStore.PutBlockHeader(header.Hash(), header, header.Difficulty, true)

	genesis := GetGenesis(GenesisInfo{})
	err := genesis.InitializeAndValidate(bcStore, db)
	assert.Equal(t, err, ErrGenesisHashMismatch)
}
