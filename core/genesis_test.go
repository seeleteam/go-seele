/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func Test_Genesis_GetGenesis(t *testing.T) {
	// case 1
	genesis1 := GetGenesis(&GenesisInfo{CreateTimestamp: big.NewInt(0)})
	genesis2 := GetGenesis(&GenesisInfo{CreateTimestamp: big.NewInt(0)})
	assert.Equal(t, genesis1.header, genesis2.header)
	assert.Equal(t, genesis1.info, NewGenesisInfo(nil, 1, 0, big.NewInt(0), types.PowConsensus, nil))
	assert.Equal(t, genesis2.info, NewGenesisInfo(nil, 1, 0, big.NewInt(0), types.PowConsensus, nil))
	validateGenesisDefaultMembers(t, genesis1)
	validateGenesisDefaultMembers(t, genesis2)

	// case 2
	addr := crypto.MustGenerateRandomAddress()
	accounts := make(map[common.Address]*big.Int)
	accounts[*addr] = big.NewInt(10)
	genesis3 := GetGenesis(NewGenesisInfo(accounts, 1, 0, big.NewInt(0), types.PowConsensus, nil))
	if genesis3.header.StateHash == common.EmptyHash {
		panic("genesis3 state hash should not equal to empty hash")
	}

	if genesis3.header == genesis2.header {
		panic("genesis3 should not equal to genesis2")
	}

	assert.Equal(t, genesis3.info, NewGenesisInfo(accounts, 1, 0, big.NewInt(0), types.PowConsensus, nil))
	validateGenesisDefaultMembers(t, genesis3)

	// case 3
	var difficult int64
	genesis4 := GetGenesis(NewGenesisInfo(accounts, difficult, 0, big.NewInt(0), types.PowConsensus, nil))
	assert.Equal(t, genesis4.header.Difficulty, big.NewInt(1))
	assert.Equal(t, genesis4.header.Witness, common.SerializePanic(shardInfo{ShardNumber: 0}))
	assert.Equal(t, genesis4.info, NewGenesisInfo(accounts, 1, 0, big.NewInt(0), types.PowConsensus, nil))
	validateGenesisDefaultMembers(t, genesis4)

	difficult = 10
	genesis4 = GetGenesis(NewGenesisInfo(nil, difficult, 0, big.NewInt(0), types.PowConsensus, nil))
	assert.Equal(t, genesis4.header.Difficulty, big.NewInt(difficult))
	assert.Equal(t, genesis4.header.Witness, common.SerializePanic(shardInfo{ShardNumber: 0}))
	assert.Equal(t, genesis4.info, NewGenesisInfo(nil, difficult, 0, big.NewInt(0), types.PowConsensus, nil))
	validateGenesisDefaultMembers(t, genesis4)

	// case 4
	var shardNumber uint = 1
	genesis5 := GetGenesis(NewGenesisInfo(nil, difficult, shardNumber, big.NewInt(0), types.PowConsensus, nil))
	assert.Equal(t, genesis5.header.Difficulty, big.NewInt(difficult))
	assert.Equal(t, genesis5.header.Witness, common.SerializePanic(shardInfo{ShardNumber: shardNumber}))
	assert.Equal(t, genesis5.info, NewGenesisInfo(nil, difficult, shardNumber, big.NewInt(0), types.PowConsensus, nil))
	validateGenesisDefaultMembers(t, genesis5)
}

func Test_Genesis_GetShardNumber(t *testing.T) {
	genesis := GetGenesis(&GenesisInfo{})
	assert.Equal(t, genesis.GetShardNumber(), uint(0))

	genesis = GetGenesis(NewGenesisInfo(nil, 0, 10, big.NewInt(0), types.PowConsensus, nil))
	assert.Equal(t, genesis.GetShardNumber(), uint(10))
}

func Test_Genesis_Init_DefaultGenesis(t *testing.T) {
	db, dispose := leveldb.NewTestDatabase()
	defer dispose()

	bcStore := store.NewBlockchainDatabase(db)

	genesis := GetGenesis(&GenesisInfo{})
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

	header := GetGenesis(&GenesisInfo{}).header.Clone()
	header.Difficulty = big.NewInt(10)
	bcStore.PutBlockHeader(header.Hash(), header, header.Difficulty, true)

	genesis := GetGenesis(&GenesisInfo{})
	err := genesis.InitializeAndValidate(bcStore, db)
	assert.Equal(t, err, ErrGenesisHashMismatch)
}

func validateGenesisDefaultMembers(t *testing.T, genesis *Genesis) {
	assert.Equal(t, genesis.header.PreviousBlockHash, common.EmptyHash)
	assert.Equal(t, genesis.header.Creator, common.EmptyAddress)
	assert.Equal(t, genesis.header.TxHash, types.MerkleRootHash(nil))
	assert.Equal(t, genesis.header.Height, genesisBlockHeight)
	assert.Equal(t, genesis.header.CreateTimestamp, big.NewInt(0))
}
