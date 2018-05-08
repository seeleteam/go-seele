/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"math"
	"math/big"
	"sync"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner/pow"
)

var logger = log.GetLogger("test", true)

func getTask(difficult int64) *Task {
	return &Task{
		header: &types.BlockHeader{
			Difficulty: big.NewInt(difficult),
		},
	}
}

func Test_Worker(t *testing.T) {
	task := getTask(10)

	result := make(chan *Result, 1)
	abort := make(chan struct{}, 1)
	isNonceFound := new(int32)

	go StartMining(task, 0, 0, math.MaxUint64, result, abort, isNonceFound, logger)

	select {
	case found := <-result:
		target := pow.GetMiningTarget(task.header.Difficulty)

		assert.Equal(t, found.task, task)

		hash := found.block.Header.Hash()
		var hashInt big.Int
		hashInt.SetBytes(hash.Bytes())
		assert.Equal(t, hashInt.Cmp(target) <= 0, true)
	}
}

func Test_WorkerStop(t *testing.T) {
	task := getTask(20)

	result := make(chan *Result, 1)
	abort := make(chan struct{}, 1)
	isNonceFound := new(int32)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		StartMining(task, 0, 0, math.MaxUint64, result, abort, isNonceFound, logger)
		wg.Done()
	}()

	close(abort)

	wg.Wait()
}
