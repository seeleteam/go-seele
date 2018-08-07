/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math/big"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/miner/pow"
)

func newTestBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           common.BytesToAddress([]byte{1}),
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            1,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Nonce:             1,
		ExtraData:         common.CopyBytes([]byte("ExtraData")),
	}
}

func Test_handleMinerRewardTx(t *testing.T) {
	db, remove := leveldb.NewTestDatabase()
	defer remove()

	statedb, err := state.NewStatedb(common.EmptyHash, db)
	if err != nil {
		panic(err)
	}

	task := getTask(10)
	task.header = newTestBlockHeader()
	reward, err := task.handleMinerRewardTx(statedb)

	assert.Equal(t, err, nil)
	assert.Equal(t, reward, pow.GetReward(task.header.Height))
}
