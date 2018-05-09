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
	metrics "github.com/rcrowley/go-metrics"
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

func NewTestMiner(t *testing.T) *Miner {
	return &Miner{
		stopChan:             make(chan struct{}, 1),
		recv:                 make(chan *Result, 1),
		log:                  logger,
		isFirstDownloader:    1,
		isFirstBlockPrepared: 0,
		isNonceFound:         new(int32),
		hashrate:             metrics.NewMeter(),
	}

}

func Test_Worker(t *testing.T) {
	task := getTask(10)

	miner := NewTestMiner(t)

	go miner.startMining(task, 0, 0, math.MaxUint64)

	select {
	case found := <-miner.recv:
		target := pow.GetMiningTarget(task.header.Difficulty)

		assert.Equal(t, found.task, task)

		hash := found.block.Header.Hash()
		var hashInt big.Int
		hashInt.SetBytes(hash.Bytes())
		assert.Equal(t, hashInt.Cmp(target) <= 0, true)
	}
}

func Test_WorkerStop(t *testing.T) {
	task := getTask(10)

	miner := NewTestMiner(t)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		miner.startMining(task, 0, 0, math.MaxUint64)
		wg.Done()
	}()

	close(miner.stopChan)

	wg.Wait()
}
