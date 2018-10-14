/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math/big"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/stretchr/testify/assert"
)

func getTask(difficult int64) *Task {
	return &Task{
		header: &types.BlockHeader{
			Difficulty: big.NewInt(difficult),
		},
	}
}

func newTestBlockHeader() *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           common.BytesToAddress([]byte{1}),
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            1,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Witness:           make([]byte, 0),
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
	assert.Equal(t, reward, consensus.GetReward(task.header.Height))
}
