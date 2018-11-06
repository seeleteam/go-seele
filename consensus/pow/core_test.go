/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"math"
	"math/big"
	"sync"
	"testing"

	"github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
)

var logger = log.GetLogger("test")

func getBlock(difficult int64) *types.Block {
	return &types.Block{
		Header: &types.BlockHeader{
			Difficulty: big.NewInt(difficult),
		},
	}
}

func Test_Worker(t *testing.T) {
	block := getBlock(10)

	result := make(chan *types.Block, 1)
	abort := make(chan struct{}, 1)
	isNonceFound := new(int32)
	hashrate := metrics.NewMeter()

	go StartMining(block, 0, 0, math.MaxUint64, result, abort, isNonceFound, &sync.Once{}, hashrate, logger)

	select {
	case found := <-result:
		target := getMiningTarget(block.Header.Difficulty)

		assert.Equal(t, found, block)

		hash := found.Header.Hash()
		var hashInt big.Int
		hashInt.SetBytes(hash.Bytes())
		assert.Equal(t, hashInt.Cmp(target) <= 0, true)
	}

	prevIsNonceFound := isNonceFound

	go func() {
		defer func() {
			close(result)
		}()

		StartMining(block, 0, 0, math.MaxUint64, result, abort, isNonceFound, &sync.Once{}, hashrate, logger)
	}()

	found, ok := <-result
	assert.Equal(t, ok, false)
	assert.Equal(t, found == nil, true)

	// exit mining as nonce is found by other threads
	assert.Equal(t, prevIsNonceFound, isNonceFound)
}

func Test_WorkerStop(t *testing.T) {
	block := getBlock(20)

	result := make(chan *types.Block, 1)
	abort := make(chan struct{}, 1)
	isNonceFound := new(int32)
	hashrate := metrics.NewMeter()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		StartMining(block, 0, 0, math.MaxUint64, result, abort, isNonceFound, &sync.Once{}, hashrate, logger)
	}()

	close(abort)

	wg.Wait()
}
